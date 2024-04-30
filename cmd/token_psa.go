package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/psa"
	"golang.org/x/oauth2"
)

func psaToken(vehicleConf config.Named) (*oauth2.Token, error) {
	cc := struct {
		User    string
		Country string
	}{}

	util.DecodeOther(vehicleConf.Other, &cc)

	if cc.User == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Country == "" {
		prompt_country := &survey.Input{
			Message: "Please enter your country code:",
		}
		if err := survey.AskOne(prompt_country, &cc.Country, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
	}
	brand := strings.ToLower(vehicleConf.Type)
	sk := psa.SettingsKey(brand, cc.User)
	cv := oauth2.GenerateVerifier()
	oc := psa.Oauth2Config(brand, strings.ToLower(cc.Country))

	authorize_url := oc.AuthCodeURL("", oauth2.S256ChallengeOption(cv))

	fmt.Println("Please visit: ", authorize_url)
	fmt.Println("And grab the authorization code like described here: https://github.com/flobz/psa_car_controller/discussions/779")

	var code string
	prompt_code := &survey.Input{
		Message: "Please enter your authorization code:",
	}
	if err := survey.AskOne(prompt_code, &code, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	client := request.NewClient(util.NewLogger(brand))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	// get token
	tok, err := oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
	if err != nil {
		return nil, err
	}

	// save token in settings db
	err = settings.SetJson(sk, tok)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println("Token successful retrieved and stored in settings DB!")

	return nil, nil
}
