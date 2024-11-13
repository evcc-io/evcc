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

func MqttCredentials(account, serial string, global bool) (CredentialsResponse, error) {
	client := request.NewHelper(util.NewLogger("zendure"))

	data := CredentialsRequest{
		SnNumber: serial,
		Account:  account,
	}

	uri := EUCredentialsUri
	if global {
		uri = GlobalCredentialsUri
	}

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	var res CredentialsResponse
	err := client.DoJSON(req, &res)

	if err == nil && !res.Success {
		err = errors.New(res.Msg)
	}

	return res, err
}
