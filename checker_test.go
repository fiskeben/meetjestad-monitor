package main

import (
	"context"
	"errors"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestCompareSensorData(t *testing.T) {
	type args struct {
		sensor  Sensor
		reading Reading
		alarm   Alarm
	}

	nowFunc = func() time.Time {
		return time.Date(2019, 7, 3, 23, 12, 45, 123, time.UTC)
	}

	okPos := Position{Lat: 1.23, Lng: 3.21}

	tenHoursAgo := time.Duration(-36000000000000)

	tests := []struct {
		name string
		args args
		want Alarm
	}{
		{
			name: "ok data gives no alarms",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 3.1, Date: nowFunc(), Position: okPos},
			},
		},
		{
			name: "raises offline alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 3.1, Date: nowFunc().Add(tenHoursAgo)},
			},
			want: Alarm{Offline: nowFunc()},
		},
		{
			name: "raises gps alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 3.1, Date: nowFunc()},
			},
			want: Alarm{GpsMissing: nowFunc()},
		},
		{
			name: "raises voltage alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 2.9, Date: nowFunc(), Position: okPos},
			},
			want: Alarm{LowVoltage: nowFunc()},
		},
		{
			name: "does not re-check offline alarm",
			args: args{
				sensor:  Sensor{Threshold: 3.2, Alarms: Alarm{Offline: nowFunc().Add(tenHoursAgo)}},
				reading: Reading{Voltage: 3.1, Date: nowFunc().Add(tenHoursAgo)},
			},
			want: Alarm{Offline: nowFunc().Add(tenHoursAgo)},
		},
		{
			name: "has battery alarm checks gps",
			args: args{
				sensor:  Sensor{Threshold: 3.2, Alarms: Alarm{LowVoltage: nowFunc().Add(tenHoursAgo)}},
				reading: Reading{Voltage: 3.1, Date: nowFunc()},
			},
			want: Alarm{LowVoltage: nowFunc().Add(tenHoursAgo), GpsMissing: nowFunc()},
		},
	}

	for _, tt := range tests {
		a := compareSensorData(tt.args.sensor, tt.args.reading)
		//t.Logf("res %v", a)
		if diff := deep.Equal(a, tt.want); diff != nil {
			t.Errorf("%s failed: %v", tt.name, diff)
		}
	}
}

type sensorsMock struct {
	mock.Mock
}

func (sm *sensorsMock) Next(ctx context.Context, s *Sensor) error {
	args := sm.Called(ctx, s)
	if len(sm.Calls) >= 2 {
		return ErrSensorEOF
	}

	sensor := args.Get(0).(Sensor)
	s.ID = sensor.ID
	s.Threshold = sensor.Threshold
	s.EmailAddress = sensor.EmailAddress
	return args.Error(1)
}

func (sm *sensorsMock) Stop() {
	sm.Called()
}

func (sm *sensorsMock) Store(ctx context.Context, s Sensor) error {
	args := sm.Called(ctx, s)
	return args.Error(0)
}

type sensorReaderMock struct {
	mock.Mock
}

func (sgm *sensorReaderMock) Read(r *Reading) error {
	args := sgm.Called(r)
	return args.Error(0)
}

func TestCheckSensors(t *testing.T) {
	nowFunc = func() time.Time {
		return time.Date(2019, 7, 3, 23, 12, 45, 123, time.UTC)
	}

	type args struct {
		m       Mailer
		c       sensorReader
		sensors *sensorsMock
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "sends mail",
			args: args{
				m: &logMailer{},
				c: func() *sensorReaderMock {
					s := sensorReaderMock{}
					s.On("Read", &Reading{SensorID: "123"}).Return(nil)
					return &s
				}(),
				sensors: func() *sensorsMock {
					s := sensorsMock{}
					s.On("Next", context.Background(), &Sensor{}).Return(Sensor{ID: "123"}, nil)
					s.On("Stop").Once().Return()
					s.On("Store", context.Background(), Sensor{ID: "123", Alarms: Alarm{Offline: nowFunc()}}).Return(nil)
					return &s
				}(),
			},
		},
		{
			name: "fails to get next sensor",
			args: args{
				m: &logMailer{},
				sensors: func() *sensorsMock {
					s := sensorsMock{}
					s.On("Next", context.Background(), &Sensor{}).Return(Sensor{}, errors.New("test error"))
					s.On("Stop").Once().Return()
					return &s
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Logf("executing '%s'", tt.name)
		err := checkSensors(tt.args.m, tt.args.c, tt.args.sensors)
		if err != nil && !tt.wantErr {
			t.Errorf("%s failed: %v", tt.name, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s expected error", tt.name)
		}
		tt.args.sensors.AssertExpectations(t)
	}
}
