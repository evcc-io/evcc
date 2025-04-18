package cmd

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/evcc-io/evcc/vehicle/volvo/connected"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

func volvoToken(conf config.Named) (*oauth2.Token, error) {
	var cc struct {
		Credentials vehicle.ClientCredentials
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

	oc := connected.Oauth2Config(cc.Credentials.ID, cc.Credentials.Secret)
	cv := oauth2.GenerateVerifier()

	state := lo.RandomString(16, lo.AlphanumericCharset)
	authorize_url := oc.AuthCodeURL(state, oauth2.S256ChallengeOption(cv))

	fmt.Println("Please visit: ", authorize_url)
	fmt.Println("And grab the authorization code like described here: https://github.com/flobz/psa_car_controller/discussions/779")

	var code string
	prompt_code := &survey.Input{
		Message: "Please enter your authorization code:",
	}
	if err := survey.AskOne(prompt_code, &code, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	client := request.NewClient(util.NewLogger("volvo-connected"))
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
