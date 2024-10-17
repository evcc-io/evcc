package cmd

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/evcc-io/evcc/vehicle/ford/connect"
	"golang.org/x/oauth2"
)

func fordConnectToken(conf config.Named) (*oauth2.Token, error) {
	var cc struct {
		Credentials vehicle.ClientCredentials
		Tokens      vehicle.Tokens
		Other       map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(conf.Other, &cc); err != nil {
		return nil, err
	}

	if cc.Credentials.ID == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Please enter your client id:",
		}, &cc.Credentials.ID, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
	}

	if cc.Credentials.Secret == "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Please enter your client secret:",
		}, &cc.Credentials.Secret, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}
	}

	var code string
	if err := survey.AskOne(&survey.Input{
		Message: "Please enter your authorization code:",
	}, &code, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	cv := oauth2.GenerateVerifier()
	oc := connect.Oauth2Config(cc.Credentials.ID, cc.Credentials.Secret)

	client := request.NewClient(util.NewLogger("ford-connect"))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
