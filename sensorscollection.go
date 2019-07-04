package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"google.golang.org/api/iterator"
)

var ErrSensorEOF = errors.New("no more sensors")

type SensorIteratable interface {
	Next(ctx context.Context, s *Sensor) error
	Stop()
}

type SensorCollection struct {
	collection *firestore.CollectionRef
	iterator   *firestore.DocumentIterator
}

func (s *SensorCollection) Next(ctx context.Context, sensor *Sensor) error {
	if s.iterator == nil {
		s.iterator = s.collection.Documents(ctx)
	}

	snapshot, err := s.iterator.Next()
	if err != nil {
		if err == iterator.Done {
			return ErrSensorEOF
		}
		return err
	}

	if err := snapshot.DataTo(&sensor); err != nil {
		return err
	}

	return nil
}

func (s *SensorCollection) Stop() {
	if s.iterator != nil {
		s.iterator.Stop()
	}
}
