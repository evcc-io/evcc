package keys

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

const keyStore = "https://renault-wrd-prod-1-euw1-myrapp-one.s3-eu-west-1.amazonaws.com/configuration/android/config_%s.json"

type configResponse struct {
	Servers configServers
}

type configServers struct {
	GigyaProd ConfigServer `json:"gigyaProd"`
	WiredProd ConfigServer `json:"wiredProd"`
}

type ConfigServer struct {
	Target string `json:"target"`
	APIKey string `json:"apikey"`
}

type Keys struct {
	*request.Helper
	Gigya, Kamereon ConfigServer
}

func New(log *util.Logger) *Keys {
	return &Keys{
		Helper: request.NewHelper(log),
	}
}

func (v *Keys) Load(region string) {
	uri := fmt.Sprintf(keyStore, region)

	var cr configResponse
	if err := v.GetJSON(uri, &cr); err == nil {
		v.Gigya = cr.Servers.GigyaProd
		v.Kamereon = cr.Servers.WiredProd
		// temporary fix of wrong kamereon APIKey in keyStore
		v.Kamereon.APIKey = "VAX7XYKGfa92yMvXculCkEFyfZbuM7Ss"
	} else {
		// use old fixed keys if keyStore is not accessible
		v.Gigya = ConfigServer{"https://accounts.eu1.gigya.com", "3_7PLksOyBRkHv126x5WhHb-5pqC1qFR8pQjxSeLB6nhAnPERTUlwnYoznHSxwX668"}
		v.Kamereon = ConfigServer{"https://api-wired-prod-1-euw1.wrd-aws.com", "VAX7XYKGfa92yMvXculCkEFyfZbuM7Ss"}
	}
}
