package zendure

import (
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const CredentialsUri = "https://app.zendure.tech/eu/developer/api/apply"

func MqttCredentials(account, serial string) (CredentialsResponse, error) {
	client := request.NewHelper(util.NewLogger("zendure"))

	data := CredentialsRequest{
		SnNumber: serial,
		Account:  account,
	}

	req, _ := request.New(http.MethodPost, CredentialsUri, request.MarshalJSON(data), request.JSONEncoding)

	var res CredentialsResponse
	err := client.DoJSON(req, &res)

	if err == nil && !res.Success {
		err = errors.New(res.Msg)
	}

	return res, err
}
