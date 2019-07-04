package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AlarmGetter interface {
	Get(ctx context.Context, ID string) (Alarm, error)
}

type AlarmStorer interface {
	Store(ctx context.Context, ID string, a Alarm) error
}

type AlarmGetterStorer interface {
	AlarmGetter
	AlarmStorer
}

type AlarmsCollection struct {
	collection *firestore.CollectionRef
}

func (a *AlarmsCollection) Get(ctx context.Context, ID string) (Alarm, error) {
	var res Alarm
	doc, err := a.collection.Doc(ID).Get(ctx)
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

func (a *AlarmsCollection) Store(ctx context.Context, ID string, alarm Alarm) error {
	doc := a.collection.Doc(ID)
	_, err := doc.Set(ctx, alarm)
	if err != nil {
		return err
	}
	return nil
}
