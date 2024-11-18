package zendure

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const (
	EUCredentialsUri     = "https://app.zendure.tech/eu/developer/api/apply"
	GlobalCredentialsUri = "https://app.zendure.tech/v2/developer/api/apply"
)

func MqttCredentials(log *util.Logger, region, account, serial string) (CredentialsResponse, error) {
	client := request.NewHelper(log)

	data := CredentialsRequest{
		SnNumber: serial,
		Account:  account,
	}

	uri := GlobalCredentialsUri
	if region == "EU" {
		uri = EUCredentialsUri
	}

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var res CredentialsResponse
	err := client.DoJSON(req, &res)

	if err == nil && !res.Success {
		err = errors.New(res.Msg)
	}

	return res, err
}
