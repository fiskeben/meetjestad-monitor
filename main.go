package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	firebase "firebase.google.com/go"

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

	sc := SensorCollection{collection: fs.Collection("sensors")}
	sr := httpSensorReader{client: http.DefaultClient}

	m, err := newMailer(config.Mailer.SecretPath)
	if err != nil {
		log.Fatalln(err)
	}

	// check all sensors at start, otherwise it will wait until the first tick
	if err := checkSensors(m, &sr, &sc); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(config.Frequency)

	// listen to the tick and check the sensors
	log.Printf("starting job every %v seconds", config.Frequency)
	for {
		select {
		case <-ticker.C:
			if err := checkSensors(m, &sr, &sc); err != nil {
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
