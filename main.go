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
	Position Position  `json:"coordinates"`
}

// Position is a coordinate with latitude and longitude.
type Position struct {
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
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
	offline    time.Time
	gpsMissing time.Time
	lowVoltage time.Time
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

	alarms := make(map[string]Alarm)

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

func checkSensors(m mailer, subscriptions []Subscription, threshold float32, alarms map[string]Alarm) error {
	log.Printf("checking %d sensors", len(subscriptions))
	now := time.Now()
	for _, s := range subscriptions {
		log.Printf("checking %s", s.SensorID)
		r, err := readSensor(s.SensorID)
		if err != nil {
			return err
		}

		log.Printf("sensor data %v", r)

		alarm, ok := alarms[r.SensorID]
		if !ok {
			alarm = Alarm{}
		}

		if diff := now.Sub(r.Date); diff.Hours() > 6 && alarm.offline.IsZero() {
			log.Printf("sensor is offline for %v", diff)
			if err := raiseOutageAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			alarm.offline = now
			continue // No need to check the rest and potentially spam the recipient
		}

		if r.Voltage < threshold && alarm.lowVoltage.IsZero() {
			log.Printf("voltage is below threshold: %v < %v", r.Voltage, threshold)
			if err := raiseVoltageAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			alarm.lowVoltage = now
		}

		if r.Position.Lat == 0 && r.Position.Lng == 0 && alarm.gpsMissing.IsZero() {
			log.Printf("sensor is missing GPS lock: %v", r.Position)
			if err := raiseGPSmissingAlarm(m, s.SensorID, s.EmailAddress, r.Date); err != nil {
				return fmt.Errorf("failed to send mail: %v", err)
			}
			alarm.gpsMissing = now
		}
		alarms[r.SensorID] = alarm
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
