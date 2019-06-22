package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	firebase "firebase.google.com/go"
	"cloud.google.com/go/firestore"

	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

func main() {
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

	m, err := newMailer()
	if err != nil {
		log.Fatalln(err)
	}

	// check all sensors at start, otherwise it will wait until the first tick
	if err := checkSensors(m, alarms, config.Subscriptions, config.Service.Threshold); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(config.Service.Frequency)

	// listen to the tick and check the sensors
	for {
		select {
		case <-ticker.C:
			// re-reading the config allows us to update the list of subscriptions without restarting
			var err error
			config, err = readConfig()
			if err != nil {
				log.Printf("config error, unable to continue, please fix the error first: %v", err)
				continue
			}
			if err := checkSensors(m, alarms, config.Subscriptions, config.Service.Threshold); err != nil {
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

	if c.Service.Threshold == 0 {
		c.Service.Threshold = defaultConfig.Service.Threshold
	}
	if c.Service.Frequency == 0 {
		c.Service.Frequency = defaultConfig.Service.Frequency
	}
	if c.Service.Mailer.Domain == "" {
		c.Service.Mailer.Domain = defaultConfig.Service.Mailer.Domain
	}
	if c.Subscriptions == nil {
		c.Subscriptions = make([]Subscription, 0, 0)
	}

	return c, nil
}

func checkSensors(m mailer, collection *firestore.CollectionRef, subscriptions []Subscription, threshold float32) error {
	log.Printf("checking %d sensors", len(subscriptions))
	now := time.Now()
	ctx := context.Background()

	for _, s := range subscriptions {
		log.Printf("checking %s", s.SensorID)
		r, err := readSensor(s.SensorID)
		if err != nil {
			return err
		}

		log.Printf("sensor data %v", r)

		a, err := alarmsForSensor(ctx, collection, s.SensorID)
		if err != nil {
			log.Printf("error getting alarms for %s: %v", s.SensorID, err)
			continue
		}

		age := now.Sub(a.Offline)
		if diff := now.Sub(r.Date); diff.Hours() > 6 && age.Hours() > 24 {
			log.Printf("sensor is offline for %v", diff)
			if err := raiseOutageAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.Offline = now

			// No need to check the rest and potentially spam the recipient
			if err = storeSensorAlarms(ctx, collection, s.SensorID, a); err != nil {
				log.Printf("failed to store alarm for sensorr %s: %v", s.SensorID, err)
			}
			continue
		} else {
			a.Offline = time.Time{}
		}

		age = now.Sub(a.LowVoltage)
		log.Println("low voltage age", age, now, a.LowVoltage)
		if r.Voltage < threshold && age > 24 {
			log.Printf("voltage is below threshold: %v < %v", r.Voltage, threshold)
			if err := raiseVoltageAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.LowVoltage = now
		} else {
			a.LowVoltage = time.Time{}
		}

		age = now.Sub(a.GpsMissing)
		if r.Position.Lat == 0 && r.Position.Lng == 0 && age.Hours() > 24 {
			log.Printf("sensor is missing GPS lock: %v", r.Position)
			if err := raiseGPSmissingAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			a.GpsMissing = now
		} else {
			a.GpsMissing = time.Time{}
		}
		if err = storeSensorAlarms(ctx, collection, s.SensorID, a); err != nil {
			log.Printf("failed to store alarm for sensorr %s: %v", s.SensorID, err)
		}
	}

	return nil
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
