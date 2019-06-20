package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go"
)

func getAlarms(c *firebase.App) error {
	ctx := context.Background()

	f, err := c.Firestore(ctx)
	if err != nil {
		return err
	}

	alarms := f.Collection("alarms")

	log.Printf("alarms collection: %v", alarms)
	return nil
}
