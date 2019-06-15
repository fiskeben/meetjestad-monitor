package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"
)

// Reading represents one unique data point.
type Reading struct {
	SensorID string    `json:"sensor_id"`
	Date     time.Time `json:"date"`
	Voltage  float32   `json:"voltage"`
	Firmware string    `json:"firmware_version"`
}

// Subscription represents a sensor to monitor and an email address to send alarms to.
type Subscription struct {
	SensorID     string
	EmailAddress string
}

const (
	threshold = 3.25
)

func main() {
	// check all sensors at start, otherwise it will wait until the first tick
	if err := checkSensors(); err != nil {
		panic(err)
	}

	d, err := time.ParseDuration("1h")
	if err != nil {
		panic(err)
	}
	ticker := time.NewTicker(d)

	// listen to the tick and check the sensors
	for {
		select {
		case <-ticker.C:
			if err := checkSensors(); err != nil {
				panic(err)
			}
		}
	}
}

func readConfig() ([]Subscription, error) {
	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	var r []Subscription
	if err = yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}

	return r, nil
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

	log.Println("got response")
	defer res.Body.Close()
	var data []Reading
	d := json.NewDecoder(res.Body)
	if err := d.Decode(&data); err != nil {
		return Reading{}, err
	}

	if len(data) == 0 {
		return Reading{Voltage: 999}, nil
	}

	r := data[0]
	log.Printf("decoded response: %v", r)
	return r, nil
}

func checkSensors() error {
	// read the config every time. This allows adding more sensors without restarting.
	subscriptions, err := readConfig()
	if err != nil {
		return err
	}

	for _, s := range subscriptions {
		r, err := readSensor(s.SensorID)
		if err != nil {
			return err
		}
		if r.Voltage < threshold {
			log.Printf("voltage is below threshold: %v", r.Voltage)
			if err := sendMail(s.SensorID, s.EmailAddress, r.Date); err != nil {
				return err
			}
		}
	}

	return nil
}
