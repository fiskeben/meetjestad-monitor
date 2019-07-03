package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v3"
)

type mailer struct {
	mg mailgun.Mailgun
}

type logMailer struct {
	mailgun.Mailgun
	instance *mailgun.MailgunImpl
}

func (l logMailer) NewMessage(from, subject, text string, to ...string) *mailgun.Message {
	return l.instance.NewMessage(from, subject, text, to...)
}

func (l logMailer) Send(ctx context.Context, m *mailgun.Message) (string, string, error) {
	log.Println("sending dummy mail", m)
	return "ok", "ok", nil
}

const (
	domain = "monitoring.meetjescraper.online"
)

func newMailer(path string) (mailer, error) {
	if path == "" {
		return newDummyMailer()
	}

	var mg mailgun.Mailgun

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return mailer{}, fmt.Errorf("unable to read Mailgun secrets file: %v", err)
	}
	apiKey := string(b)
	mg = mailgun.NewMailgun(domain, apiKey)
	mg.SetAPIBase("https://api.eu.mailgun.net/v3")

	return mailer{mg: mg}, nil
}

func newDummyMailer() (mailer, error) {
	log.Println("using dummy mailer: printing mails to log")
	mg := logMailer{instance: mailgun.NewMailgun("", "")}

	return mailer{mg: mg}, nil
}

func raiseVoltageAlarm(m mailer, sensorID, recipient string, lastSeen time.Time) error {
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

The Meetjestad monitoring robot`, sensorID, formattedDate)

	return sendMail(m, recipient, sender, subject, body)
}

func raiseOutageAlarm(m mailer, sensorID, recipient string, lastSeen time.Time) error {
	formattedDate := lastSeen.Format(time.RFC822)

	sender := "alert@monitoring.meetjescraper.online"
	subject := "Meetjestad station is offline"
	body := fmt.Sprintf(`Hi,

This is an automated message to tell you that your sensor with ID %s seems to be offline.

It was last seen at %s.

-- 
Regards,

The Meetjestad monitoring robot`, sensorID, formattedDate)

	return sendMail(m, recipient, sender, subject, body)
}

func raiseGPSmissingAlarm(m mailer, sensorID, recipient string, lastSeen time.Time) error {
	formattedDate := lastSeen.Format(time.RFC822)

	sender := "alert@monitoring.meetjescraper.online"
	subject := "Meetjestad station is missing GPS lock"
	body := fmt.Sprintf(`Hi,

This is an automated message to tell you that your sensor with ID %s is missing GPS lock.

This applies to the latest message which was received at %s.

You should make sure that your weather station has a clear view of the sky
and perhaps also attempt to reset it while outdoors.

-- 
Regards,

The Meetjestad monitoring robot`, sensorID, formattedDate)

	return sendMail(m, recipient, sender, subject, body)
}

func sendMail(m mailer, to, from, subject, body string) error {
	message := m.mg.NewMessage(from, subject, body, to)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, id, err := m.mg.Send(ctx, message)

	if err != nil {
		return err
	}

	log.Printf("mail sent: ID: %s Resp: %s\n", id, resp)

	return nil
}
