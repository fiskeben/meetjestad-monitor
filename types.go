package main

import "time"

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
	Frequency time.Duration
	Mailer    MailerConfig
}

// MailerConfig stores configuration for Mailgun.
type MailerConfig struct {
	SecretPath string `yaml:"secretPath"`
	Domain     string
	APIBase    string
}

// Subscription represents a sensor to monitor and an email address to send alarms to.
type Sensor struct {
	ID           string  `firestore:"sensor_id"`
	EmailAddress string  `firestore:"email_address"`
	Threshold    float32 `firestore:"threshold"`
	Owner        string  `firestore:"owner"`
	Alarms       Alarm   `firestore:"alarms"`
	DocumentID   string
}

// Alarm represents a sensor that was below the threshold and an email has been sent.
type Alarm struct {
	Offline    time.Time `firestore:"offline"`
	GpsMissing time.Time `firestore:"gps"`
	LowVoltage time.Time `firestore:"voltage"`
}

var defaultConfig = Config{
	Frequency: time.Duration(3600000000000),
	Mailer: MailerConfig{
		Domain:  "monitoring.meetjescraper.online",
		APIBase: "https://api.eu.mailgun.net/v3",
	},
}
