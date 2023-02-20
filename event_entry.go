package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type JsonClientTime time.Time

func (j *JsonClientTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	// time.DateTime constant with layout could be used in Go 1.20+
	// ref: https://cs.opensource.google/go/go/+/refs/tags/go1.20:src/time/format.go;l=119
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*j = JsonClientTime(t)
	return nil
}

func (j *JsonClientTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*j))
}

func (j *JsonClientTime) ToTime() time.Time {
	return time.Time(*j)
}

type eventEntry struct {
	ClientTime *JsonClientTime `json:"client_time" analytics:"required"`
	DeviceId   *string         `json:"device_id" analytics:"required"`
	DeviceOs   *string         `json:"device_os" analytics:"required"`
	Session    *string         `json:"session" analytics:"required"`
	Sequence   *int            `json:"sequence" analytics:"required"`
	Event      *string         `json:"event" analytics:"required"`
	ParamInt   *int            `json:"param_int" analytics:"required"`
	ParamStr   *string         `json:"param_str" analytics:"required"`
	ip         string
	serverTime time.Time
}

func (e *eventEntry) enrichData(clientAddr string, serverTime time.Time) {
	e.ip = clientAddr
	e.serverTime = serverTime
}

func (e *eventEntry) checkRequiredFields() error {
	fields := reflect.ValueOf(e).Elem()
	for i := 0; i < fields.NumField(); i++ {
		analyticsTags := fields.Type().Field(i).Tag.Get("analytics")
		if strings.Contains(analyticsTags, "required") && fields.Field(i).IsNil() {
			return errors.New(fmt.Sprintf("required field \"%s\" is missing", fields.Type().Field(i).Name))
		}
	}

	return nil
}
