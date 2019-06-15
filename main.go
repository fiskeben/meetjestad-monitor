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

// Config holds the configuration for the service.
type Config struct {
	Service       Service
	Subscriptions []Subscription
}

// Service stores the configurations for the service itself.
type Service struct {
	Threshold float32
	Frequency time.Duration
	Mailer    MailerConfig
}

// MailerConfig stores configuration for Mailgun.
type MailerConfig struct {
	Domain  string
	APIBase string
}

// Subscription represents a sensor to monitor and an email address to send alarms to.
type Subscription struct {
	SensorID     string
	EmailAddress string
}

// Alarm represents a sensor that was below the threshold and an email has been sent.
type Alarm struct {
	SensorID string
	RaisedAt time.Time
}

var defaultConfig = Config{
	Service: Service{
		Threshold: 3.33,
		Frequency: time.Duration(3600000000000),
		Mailer: MailerConfig{
			Domain:  "monitoring.meetjescraper.online",
			APIBase: "https://api.eu.mailgun.net/v3",
		},
	},
	Subscriptions: make([]Subscription, 0, 0),
}

func main() {
	config, err := readConfig()
	if err != nil {
		panic(err)
	}

	alarms := make([]Alarm, 0, 10)

	m, err := newMailer()
	if err != nil {
		panic(err)
	}

	// check all sensors at start, otherwise it will wait until the first tick
	if err := checkSensors(m, config.Subscriptions, config.Service.Threshold, alarms); err != nil {
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
			if err := checkSensors(m, config.Subscriptions, config.Service.Threshold, alarms); err != nil {
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

func checkSensors(m mailer, subscriptions []Subscription, threshold float32, alarms []Alarm) error {
	for _, s := range subscriptions {
		r, err := readSensor(s.SensorID)
		if err != nil {
			return err
		}
		if r.Voltage < threshold && !isAlerted(alarms, r.SensorID) {
			log.Printf("voltage is below threshold: %v < %v", r.Voltage, threshold)
			if err := sendMail(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			alarms = append(alarms, Alarm{SensorID: r.SensorID, RaisedAt: time.Now()})
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

func isAlerted(alarms []Alarm, sensorID string) bool {
	for _, a := range alarms {
		if a.SensorID == sensorID {
			return true
		}
	}
	return false
}
