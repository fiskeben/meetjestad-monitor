package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
)

var testDate = time.Date(2019, 7, 3, 23, 12, 45, 123, time.Local)

func TestCompose(t *testing.T) {
	type args struct {
		a Alarm
		r Reading
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "creates body for offline alarm",
			args: args{
				a: Alarm{Offline: testDate},
				r: Reading{Date: testDate},
			},
			want: fixture("offline"),
		},
		{
			name: "creates body for missing GPS",
			args: args{
				a: Alarm{GpsMissing: testDate},
				r: Reading{Date: testDate},
			},
			want: fixture("gpsmissing"),
		},
		{
			name: "creates body for low battery",
			args: args{
				a: Alarm{LowVoltage: testDate},
				r: Reading{Date: testDate, Voltage: 3.25},
			},
			want: fixture("lowbattery"),
		},
		{
			name: "creates body for offline and low battery",
			args: args{
				a: Alarm{Offline: testDate, LowVoltage: testDate},
				r: Reading{Date: testDate, Voltage: 3.2},
			},
			want: fixture("offline-battery"),
		},
		{
			name: "creates body for all alarms",
			args: args{
				a: Alarm{Offline: testDate, LowVoltage: testDate, GpsMissing: testDate},
				r: Reading{Date: testDate, Voltage: 3.2},
			},
			want: fixture("all-alarms"),
		},
	}

	for _, tt := range tests {
		res := compose(tt.args.a, tt.args.r)
		if diff := deep.Equal(res, tt.want); diff != nil {
			fmt.Printf("res : %v\n", []byte(res))
			fmt.Printf("want: %v\n", []byte(tt.want))
			t.Errorf("%s failed: %v", tt.name, diff)
		}
	}
}

func fixture(name string) string {
	b, err := ioutil.ReadFile("testdata/" + name + ".txt")
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}
