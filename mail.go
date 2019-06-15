package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v3"
)

type mailer struct {
	mg mailgun.Mailgun
}

const (
	domain = "monitoring.meetjescraper.online"
)

func newMailer() (mailer, error) {
	apiKey := os.Getenv("MEETJESCRAPER_MAILGUN_API_KEY")
	if apiKey == "" {
		return mailer{}, errors.New("missing Mailgun API key (set MEETJESCRAPER_MAILGUN_API_KEY environment variable")
	}

	mg := mailgun.NewMailgun(domain, apiKey)
	mg.SetAPIBase("https://api.eu.mailgun.net/v3")

	return mailer{mg: mg}, nil
}

func sendMail(m mailer, sensorID, recipient string, lastSeen time.Time) error {

	formattedDate := lastSeen.Format(time.RFC822)

	sender := "alert@monitoring.meetjescraper.online"
	subject := "Meetjestad low battery warning"
	body := fmt.Sprintf(`Hi,

This is an automated message to tell you that your sensor with ID %s is low on battery.

The last message was received at %s.

You should take action to replace the batteries as soon as possible to avoid
the sensor going offline.

-- 
Regards,

The Meetjestad battery robot`, sensorID, formattedDate)

	message := m.mg.NewMessage(sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, id, err := m.mg.Send(ctx, message)

	if err != nil {
		return err
	}

	log.Printf("mail sent: ID: %s Resp: %s\n", id, resp)

	return nil
}
