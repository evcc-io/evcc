package zero

import (
	"encoding/json"
	"fmt"
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
	Vin      string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger, user, password, vin string) *Identity {
	v := &Identity{
		Helper:   request.NewHelper(log),
		User:     user,
		Password: password,
		Vin:      vin,
	}
	return v
}

func (v *Identity) Login() error {
	var err error
	v.UnitId, err = v.retrievedeviceId()
	if err != nil {
		return err
	}

	return nil
}

func (v *Identity) retrievedeviceId() (string, error) {
	var units UnitData
	var resp *http.Response
	var err error

	params := url.Values{
		"user":        {v.User},
		"pass":        {v.Password}, // Shold be Sha1 encoded
		"unitnumber":  {v.UnitId},
		"commandname": {"get_units"},
	}

	req, err := createRequest(&params)
	if err != nil {
		return "", err
	}

	resp, err = v.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return "", err
	}

	var body *[]byte
	body, err = handleResponse(resp)
	if err != nil {
		return "", err
	} else {
		err = json.Unmarshal(*body, &units)
		if err != nil {
			return "", err
		}
	}

	if v.Vin == "" {
		return units[0].Unitnumber, nil
	}

	for _, unit := range units {
		if unit.Name == v.Vin {
			return unit.Unitnumber, nil
		}
	}
	return "", fmt.Errorf("VIN not found")
}
