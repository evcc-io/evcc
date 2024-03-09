package cmd

import (
	"errors"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/vehicle/mercedes"
	"golang.org/x/oauth2"
)

func mercedesUsernameAndRegionPrompt() (string, string, error) {
	prompt_user := &survey.Input{
		Message: "Please enter your Mercedes ME user-account (e-mail or mobile)",
	}
	var user string
	if err := survey.AskOne(prompt_user, &user, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}

	prompt_region := &survey.Select{
		Message: "Choose your MB region:",
		Options: []string{"APAC", "EMEA", "NORAM"},
	}
	var region string
	if err := survey.AskOne(prompt_region, &region, survey.WithValidator(survey.Required)); err != nil {
		return "", "", err
	}

	return user, region, nil
}

func mercedesPinPrompt() (string, error) {
	var code string
	prompt_pin := &survey.Input{
		Message: "Please enter the Pin that you received via email/sms",
	}
	if err := survey.AskOne(prompt_pin, &code, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return strings.TrimSpace(code), nil
}

func mercedesToken() (*oauth2.Token, error) {
	// Get username and region from user to initate the email process
	username, region, err := mercedesUsernameAndRegionPrompt()
	if err != nil {
		return nil, err
	}

	api := mercedes.NewSetupAPI(log, username, region)
	result, nonce, err := api.RequestPin()
	if err != nil {
		return nil, err
	}

	if result {
		pin, err := mercedesPinPrompt()
		if err != nil {
			return nil, err
		}

		token, err := api.RequestAccessToken(*nonce, pin)
		if err == nil {
			return token, nil
		}
	}

	return nil, errors.New("unknown PinResponse - 200, Email empty")
}
