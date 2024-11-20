package niu

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalNiuToken(t *testing.T) {
	var tok Token
	str := `{"data":{"token":{"access_token":"access","refresh_token":"refresh","refresh_token_expires_in":1620139071,"token_expires_in":1620139071},"user":{"user_id":"user","mobile":"","email":"email","country_code":"49","nick_name":"nickname","real_name":"realname","last_name":"lastname","identification_code":"identification_code","birthdate":"","gender":1,"avatar":"","thumb_avatar":"","profession":0,"income":0,"car_owners":0,"purpose":"","sign":"A NIU WAY FORWARD","background":"https://download.niucache.com/static/upload/20200507/57f16705-48e9-4080-8246-b2fde8df30b5.jpg","height":0,"weight":0}},"desc":"ok","status":0}`

	if err := json.Unmarshal([]byte(str), &tok); err != nil {
		t.Error(err)
	}

	if tok.AccessToken != "access" {
		t.Error("AccessToken")
	}

	if tok.RefreshToken != "refresh" {
		t.Error("RefreshToken")
	}

	if tok.Expiry.IsZero() {
		t.Error("TokenExpiresIn")
	}
}

func TestUnmarshalNiuResponse(t *testing.T) {
	var tok Response
	str := `{"data":{"isCharging":0,"lockStatus":0,"isAccOn":0,"isFortificationOn":0,"isConnected":true,"position":{"lat":42.330353,"lng":11.450598},"hdop":1,"time":1617697890173,"batteries":{"compartmentA":{"bmsId":"bmsId","isConnected":true,"batteryCharging":59,"gradeBattery":"94.8"},"compartmentB":{"bmsId":"bmsId","isConnected":true,"batteryCharging":58,"gradeBattery":"95.9"}},"leftTime":10.2,"estimatedMileage":68,"gpsTimestamp":1617697890173,"infoTimestamp":1617697890173,"nowSpeed":0,"shakingValue":3,"locationType":1,"batteryDetail":true,"centreCtrlBattery":100,"ss_protocol_ver":3,"ss_online_sta":"1","gps":5,"gsm":14,"lastTrack":{"ridingTime":90,"distance":571,"time":1617269526360}},"desc":"成功","trace":"成功","status":0}`

	if err := json.Unmarshal([]byte(str), &tok); err != nil {
		t.Error(err)
	}

	if tok.Data.Batteries.CompartmentA.BatteryCharging != 59 {
		t.Error("BatteryCharging")
	}

	if tok.Data.EstimatedMileage != 68 {
		t.Error("EstimatedMileage")
	}

	if tok.Data.IsCharging != 0 {
		t.Error("IsCharging")
	}

	// if tok.Data.LeftTime != 10.2 {
	// 	t.Error("LeftTime")
	// }
}
