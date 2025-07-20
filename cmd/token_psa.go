package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/psa"
	"golang.org/x/oauth2"
)

func psaToken(brand string) (*oauth2.Token, error) {
	var country string
	prompt_country := &survey.Input{
		Message: "Please enter your country code:",
	}
	if err := survey.AskOne(prompt_country, &country, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	cv := oauth2.GenerateVerifier()
	oc := psa.Oauth2Config(brand, strings.ToLower(country))

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

	return oc.Exchange(ctx, code, oauth2.VerifierOption(cv))
}
