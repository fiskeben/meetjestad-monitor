package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"

	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

func main() {
	var mailerSecretPath string
	flag.StringVar(&mailerSecretPath, "m", "", "path to file holding Mailgun secret")
	flag.Parse()

	ctx := context.Background()

	config, err := readConfig()
	if err != nil {
		log.Fatalln(err)
	}

	opt := option.WithCredentialsFile("serviceaccountkey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalln(err)
	}

	fs, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	alarms := fs.Collection("alarms")
	sensors := fs.Collection("sensors")

	m, err := newMailer(mailerSecretPath)
	if err != nil {
		log.Fatalln(err)
	}

	// check all sensors at start, otherwise it will wait until the first tick
	if err := checkSensors(m, alarms, sensors); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(config.Frequency)

	// listen to the tick and check the sensors
	log.Printf("starting job every %v seconds", config.Frequency)
	for {
		select {
		case <-ticker.C:
			if err := checkSensors(m, alarms, sensors); err != nil {
				log.Println(err)
			}
		}
	}
}

func readConfig() (Config, error) {
	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		return defaultConfig, err
	}

	var c Config
	if err = yaml.Unmarshal(b, &c); err != nil {
		return defaultConfig, err
	}

	if c.Frequency == 0 {
		c.Frequency = defaultConfig.Frequency
	}
	if c.Mailer.Domain == "" {
		c.Mailer.Domain = defaultConfig.Mailer.Domain
	}

	return c, nil
}

func checkSensors(m mailer, collection *firestore.CollectionRef, sensors *firestore.CollectionRef) error {
	log.Printf("checking sensors")
	now := time.Now()
	ctx := context.Background()

	it := sensors.Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				log.Println("done checking")
				return nil
			}
			return err
		}

		var s Sensor
		if err := doc.DataTo(&s); err != nil {
			return err
		}

		log.Printf("checking %s", s.ID)
		r, err := readSensor(s.ID)
		if err != nil {
			return err
		}

		log.Printf("sensor data %v", r)

		a, err := alarmsForSensor(ctx, collection, s.ID)
		if err != nil {
			log.Printf("error getting alarms for %s: %v", s.ID, err)
			continue
		}

		age := now.Sub(a.Offline)
		if diff := now.Sub(r.Date); diff.Hours() > 6 && age.Hours() > 24 {
			log.Printf("sensor is offline for %v", diff)
			if err := raiseOutageAlarm(m, s.ID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.Offline = now

			// No need to check the rest and potentially spam the recipient
			if err = storeSensorAlarms(ctx, collection, s.ID, a); err != nil {
				log.Printf("failed to store alarm for sensorr %s: %v", s.ID, err)
			}
			continue
		} else {
			a.Offline = time.Time{}
		}

		age = now.Sub(a.LowVoltage)
		threshold := s.Threshold
		if threshold == 0 {
			threshold = 3.26 // default
		}
		if r.Voltage < threshold && age > 24 {
			log.Printf("voltage is below threshold: %v < %v", r.Voltage, s.Threshold)
			if err := raiseVoltageAlarm(m, s.ID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.LowVoltage = now
		} else {
			a.LowVoltage = time.Time{}
		}

		age = now.Sub(a.GpsMissing)
		if r.Position.Lat == 0 && r.Position.Lng == 0 && age.Hours() > 24 {
			log.Printf("sensor is missing GPS lock: %v", r.Position)
			if err := raiseGPSmissingAlarm(m, s.ID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.GpsMissing = now
		} else {
			a.GpsMissing = time.Time{}
		}
		if err = storeSensorAlarms(ctx, collection, s.ID, a); err != nil {
			log.Printf("failed to store alarm for sensorr %s: %v", s.ID, err)
		}
	}
}

func readSensor(sensorID string) (Reading, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://meetjescraper.online/?sensor=%s&limit=1", sensorID), nil)
	if err != nil {
		return Reading{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Reading{}, err
	}

	defer res.Body.Close()
	var data []Reading
	d := json.NewDecoder(res.Body)
	if err := d.Decode(&data); err != nil {
		return Reading{}, err
	}

	if len(data) == 0 {
		return Reading{Voltage: 999}, nil
	}

	return data[0], nil
}
