package zero

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Identity struct {
	*request.Helper
	User     string
	Password string
	UnitId   string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger, user, password string) *Identity {
	v := &Identity{
		Helper:   request.NewHelper(log),
		User:     user,
		Password: password,
	}
	return v
}

func (v *Identity) Login() error {
	var err error
	data := url.Values{
		"user": {v.User},
		"pass": {v.Password},
	}

	v.UnitId, err = v.retrievedeviceId(data)
	if err != nil {
		return err
	}

	return nil
}

func (v *Identity) retrievedeviceId(params url.Values) (string, error) {
	var units UnitData

	var err error

	params.Set("commandname", "get_units")
	params.Set("format", "json")

	req, err := http.NewRequest("GET", BASE_URL_P, nil)
	// get charging status of vehicle
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()

	resp, err := v.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body []byte

	body, err = io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var answer ErrorAnswer
		err = json.Unmarshal(body, &answer)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf(answer.Error)
	}
	err = json.Unmarshal(body, &units)
	if err != nil {
		return "", err
	}

	return units[0].Unitnumber, err
}
