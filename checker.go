package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var nowFunc = time.Now

func checkSensors(m Mailer, alarms AlarmGetterStorer, sensors SensorIteratable) error {
	log.Printf("checking sensors")
	ctx := context.Background()

	defer sensors.Stop()

	for {
		var s Sensor
		if err := sensors.Next(ctx, &s); err != nil {
			if err == ErrSensorEOF {
				break
			}
			return err
		}

		log.Printf("checking %v", s)

		a, err := alarms.Get(ctx, s.ID)
		if err != nil {
			log.Printf("error getting alarms for %s: %v", s.ID, err)
			continue
		}

		log.Printf("checking %s", s.ID)
		r, err := readSensor(s.ID)
		if err != nil {
			log.Printf("error reading sensor, unable to monitor: %v", err)
			continue
		}

		a = compareSensorData(s, r, a)

		noTime := time.Time{}
		if a.Offline != noTime || a.LowVoltage != noTime || a.GpsMissing != noTime {
			if err = composeAndSendAlarm(ctx, m, s, a, r); err != nil {
				log.Print(err)
				continue
			}
		}
		if err = alarms.Store(ctx, s.ID, a); err != nil {
			log.Printf("failed to store alarm for sensor %s: %v", s.ID, err)
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

func compareSensorData(s Sensor, r Reading, a Alarm) Alarm {
	now := nowFunc()
	log.Printf("sensor data %v", r)

	var res Alarm

	age := now.Sub(a.Offline)
	if age.Hours() <= 24 {
		res.Offline = a.Offline
		return res
	} else if diff := now.Sub(r.Date); diff.Hours() > 6 {
		log.Printf("sensor is offline for %v", diff)
		res.Offline = now
		return res // no need to continue since the rest of the checks will fail too
	}

	age = now.Sub(a.LowVoltage)
	threshold := s.Threshold
	if threshold == 0 {
		threshold = 3.26 // default
	}
	if age.Hours() <= 24 {
		res.LowVoltage = a.LowVoltage
	} else if r.Voltage < threshold {
		log.Printf("voltage is below threshold: %v < %v", r.Voltage, s.Threshold)
		res.LowVoltage = now
	}

	age = now.Sub(a.GpsMissing)
	if age.Hours() <= 24 {
		res.GpsMissing = a.GpsMissing
	} else if r.Position.Lat == 0 && r.Position.Lng == 0 {
		log.Printf("sensor is missing GPS lock: %v", r.Position)
		res.GpsMissing = now
	}

	return res
}
