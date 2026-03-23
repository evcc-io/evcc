package jlr

import (
	"strconv"

	"github.com/evcc-io/evcc/api"
	"golang.org/x/oauth2"
)

type Token struct {
	AuthToken string `json:"authorization_token"`
	ExpiresIn int    `json:"expires_in,string"`
	oauth2.Token
}

type User struct {
	HomeMarket string `json:"homeMarket"`
	UserId     string `json:"userId"`
}

type Vehicle struct {
	UserId string `json:"userId"`
	VIN    string `json:"vin"`
	Role   string `json:"role"`
}

type VehiclesResponse struct {
	Vehicles []Vehicle
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyValueList []KeyValue

type StatusResponse struct {
	VehicleStatus struct {
		CoreStatus KeyValueList
		EvStatus   KeyValueList
	}
}

type PositionResponse struct {
	Position struct {
		Latitude        float64
		Longitude       float64
		Timestamp       string
		Speed           float64
		Heading         float64
		PositionQuality any
	}
}

type PinResponse struct {
	Token string
}

type ActionResponse struct {
	FailureDescription string
}

func (l KeyValueList) StringVal(key string) (string, error) {
	for _, el := range l {
		if el.Key == key {
			return el.Value, nil
		}
	}
	return "", api.ErrNotAvailable
}

func (l KeyValueList) FloatVal(key string) (float64, error) {
	s, err := l.StringVal(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

func (l KeyValueList) IntVal(key string) (int64, error) {
	s, err := l.StringVal(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}
