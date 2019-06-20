package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func alarmsForSensor(ctx context.Context, c *firestore.CollectionRef, sensorID string) (Alarm, error) {
	var res Alarm
	doc, err := c.Doc(sensorID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return res, nil
		}
		return res, err
	}
	if err = doc.DataTo(&res); err != nil {
		return res, err
	}
	return res, nil
}

func storeSensorAlarms(ctx context.Context, c *firestore.CollectionRef, sensorID string, a Alarm) error {
	doc := c.Doc(sensorID)
	_, err := doc.Set(ctx, a)
	if err != nil {
		return err
	}
	return nil
}
