package zero

import (
	"fmt"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Identity struct {
	*request.Helper
	user     string
	password string
	unitId   string
	vin      string
}

// NewIdentity creates SAIC identity
func NewIdentity(log *util.Logger, user, password, vin string) *Identity {
	v := &Identity{
		Helper:   request.NewHelper(log),
		user:     user,
		password: password,
		vin:      vin,
	}
	return v
}

func (v *Identity) Login() error {
	var err error
	v.unitId, err = v.retrievedeviceId()

	return err
}

func (v *Identity) retrievedeviceId() (string, error) {
	var units UnitData

	params := url.Values{
		"user":        {v.user},
		"pass":        {v.password},
		"unitnumber":  {v.unitId},
		"commandname": {"get_units"},
		"format":      {"json"},
	}

	uri := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())
	err := v.GetJSON(uri, &units)
	if err != nil {
		return "", err
	}

	if v.vin == "" {
		return units[0].Unitnumber, nil
	}

	for _, unit := range units {
		if unit.Name == v.vin {
			return unit.Unitnumber, nil
		}
	}
	return "", fmt.Errorf("VIN not found")
}
