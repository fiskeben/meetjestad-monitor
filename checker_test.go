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
		return time.Date(2019, 7, 3, 23, 12, 45, 213, time.Local)
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
				alarm:   Alarm{},
			},
		},
		{
			name: "raises offline alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 3.1, Date: nowFunc().Add(tenHoursAgo)},
				alarm:   Alarm{},
			},
			want: Alarm{Offline: nowFunc()},
		},
		{
			name: "raises gps alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 3.1, Date: nowFunc()},
				alarm:   Alarm{},
			},
			want: Alarm{GpsMissing: nowFunc()},
		},
		{
			name: "raises voltage alarm",
			args: args{
				sensor:  Sensor{Threshold: 3},
				reading: Reading{Voltage: 2.9, Date: nowFunc(), Position: okPos},
				alarm:   Alarm{},
			},
			want: Alarm{LowVoltage: nowFunc()},
		},
		{
			name: "does not re-check offline alarm",
			args: args{
				sensor:  Sensor{Threshold: 3.2},
				reading: Reading{Voltage: 3.1, Date: nowFunc().Add(tenHoursAgo)},
				alarm:   Alarm{Offline: nowFunc().Add(tenHoursAgo)},
			},
			want: Alarm{Offline: nowFunc().Add(tenHoursAgo)},
		},
		{
			name: "has battery alarm checks gps",
			args: args{
				sensor:  Sensor{Threshold: 3.2},
				reading: Reading{Voltage: 3.1, Date: nowFunc()},
				alarm:   Alarm{LowVoltage: nowFunc().Add(tenHoursAgo)},
			},
			want: Alarm{LowVoltage: nowFunc().Add(tenHoursAgo), GpsMissing: nowFunc()},
		},
	}

	for _, tt := range tests {
		a := compareSensorData(tt.args.sensor, tt.args.reading, tt.args.alarm)
		//t.Logf("res %v", a)
		if diff := deep.Equal(a, tt.want); diff != nil {
			t.Errorf("%s failed: %v", tt.name, diff)
		}
	}
}

type alarmsMock struct {
	mock.Mock
}

func (am *alarmsMock) Get(ctx context.Context, ID string) (Alarm, error) {
	args := am.Called(ctx, ID)
	return args.Get(0).(Alarm), args.Error(1)
}

func (am *alarmsMock) Store(ctx context.Context, ID string, a Alarm) error {
	args := am.Called(ctx, ID, a)
	return args.Error(0)
}

type sensorsMock struct {
	mock.Mock
}

func (sm *sensorsMock) Next(ctx context.Context, s *Sensor) error {
	args := sm.Called(ctx, s)
	if len(sm.Calls) == 2 {
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

func TestCheckSensors(t *testing.T) {
	type args struct {
		m       Mailer
		alarms  *alarmsMock
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
				alarms: func() *alarmsMock {
					a := alarmsMock{}
					a.On("Get", context.Background(), "123").Return(Alarm{}, nil)
					a.On("Store", context.Background(), "123", Alarm{}).Return(nil)
					return &a
				}(),
				sensors: func() *sensorsMock {
					s := sensorsMock{}
					s.On("Next", context.Background(), &Sensor{}).Return(Sensor{ID: "123"}, nil)
					s.On("Stop").Once().Return()
					return &s
				}(),
			},
		},
		{
			name: "fails to get next sensor",
			args: args{
				m: &logMailer{},
				alarms: func() *alarmsMock {
					a := alarmsMock{}
					return &a
				}(),
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
		err := checkSensors(tt.args.m, tt.args.alarms, tt.args.sensors)
		if err != nil && !tt.wantErr {
			t.Errorf("%s failed: %v", tt.name, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s expected error", tt.name)
		}
		tt.args.sensors.AssertExpectations(t)
		tt.args.alarms.AssertExpectations(t)
	}
}
