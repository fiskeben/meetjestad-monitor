package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v3"
)

const (
	domain = "monitoring.meetjescraper.online"
)

type Mailer interface {
	Send(ctx context.Context, to, from, subject, body string) error
}

type liveMailer struct {
	mg mailgun.Mailgun
}

func (l *liveMailer) Send(ctx context.Context, to, from, subject, body string) error {
	message := l.mg.NewMessage(from, subject, body, to)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, id, err := l.mg.Send(ctx, message)

	if err != nil {
		return err
	}

	log.Printf("mail sent: ID: %s Resp: %s\n", id, resp)

	return nil
}

type logMailer struct {
	mg mailgun.Mailgun
}

func (l *logMailer) Send(ctx context.Context, to, from, subject, body string) error {
	log.Printf("sending dummy mail to=%s from=%s subject=%s", to, from, subject)
	return nil
}

func newMailer(path string) (Mailer, error) {
	if path == "" {
		return newDummyMailer()
	}

	var mg mailgun.Mailgun

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read Mailgun secrets file: %v", err)
	}
	apiKey := string(b)
	mg = mailgun.NewMailgun(domain, apiKey)
	mg.SetAPIBase("https://api.eu.mailgun.net/v3")

	return &liveMailer{mg: mg}, nil
}

func newDummyMailer() (*logMailer, error) {
	log.Println("using dummy mailer: printing mails to log")
	return &logMailer{}, nil
}

func composeAndSendAlarm(ctx context.Context, m Mailer, sensor Sensor, a Alarm, r Reading) error {
	sender := "alert@monitoring.meetjescraper.online"
	subject := "Issues with Meet je stad sensor " + sensor.ID
	body := compose(a, r)

	return m.Send(ctx, sensor.EmailAddress, sender, subject, body)
}

func compose(a Alarm, r Reading) string {
	var t time.Time

	sb := strings.Builder{}
	sb.WriteString("Hi,\n\n")
	sb.WriteString("This is an automated message to tell you that there is one or more problems with your Meet je stad weather sensor.\n\n")
	sb.WriteString("The problems are:\n\n")

	if a.Offline != t {
		formattedDate := r.Date.Format(time.RFC822)
		sb.WriteString(fmt.Sprintf("* The sensor has been offline since %s\n", formattedDate))
	}
	if a.LowVoltage != t {
		sb.WriteString(fmt.Sprintf("* The battery seems to be low: %.2fV\n", r.Voltage))
	}
	if a.GpsMissing != t {
		sb.WriteString("* The sensor has lost GPS fix\n")
	}

	sb.WriteString("\n-- \nRegards,\n\nThe Meet je stad monitoring robot")

	return sb.String()
}
