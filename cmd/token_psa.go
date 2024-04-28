package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/evcc-io/evcc/vehicle/psa"
	"golang.org/x/oauth2"
)

// type typedata struct {
// 	scheme       string
// 	realm        string
// 	clientid     string
// 	clientsecret string
// }

// func getBlub(vehicletype string) (*typedata, error) {
// 	switch vehicletype {
// 	case "citroen":
// 		return &typedata{
// 			scheme:       "mymacsdk",
// 			realm:        "citroen.com",
// 			clientid:     "5364defc-80e6-447b-bec6-4af8d1542cae",
// 			clientsecret: "iE0cD8bB0yJ0dS6rO3nN1hI2wU7uA5xR4gP7lD6vM0oH0nS8dN",
// 		}, nil
// 	case "ds":
// 		return &typedata{
// 			scheme:       "mymdssdk",
// 			realm:        "driveds.com",
// 			clientid:     "cbf74ee7-a303-4c3d-aba3-29f5994e2dfa",
// 			clientsecret: "X6bE6yQ3tH1cG5oA6aW4fS6hK0cR0aK5yN2wE4hP8vL8oW5gU3",
// 		}, nil
// 	case "opel":
// 		return &typedata{
// 			scheme:       "mymopsdk",
// 			realm:        "opel.com",
// 			clientid:     "07364655-93cb-4194-8158-6b035ac2c24c",
// 			clientsecret: "F2kK7lC5kF5qN7tM0wT8kE3cW1dP0wC5pI6vC0sQ5iP5cN8cJ8",
// 		}, nil
// 	case "peugeot":
// 		return &typedata{
// 			scheme:       "mymap",
// 			realm:        "peugeot.com",
// 			clientid:     "1eebc2d5-5df3-459b-a624-20abfcf82530",
// 			clientsecret: "T5tP7iS0cO8sC0lA2iE2aR7gK6uE5rF3lJ8pC3nO1pR7tL8vU1",
// 		}, nil
// 	}
// 	return nil, errors.New("unsupported vehicle type")
// }

func psaToken(vehicleConf config.Named) (*oauth2.Token, error) {
	// td, err := getBlub(vehicleConf.Type)
	// if err != nil {
	// 	return nil, err
	// }

	var country string
	prompt_country := &survey.Input{
		Message: "Please enter your contry code:",
	}
	if err := survey.AskOne(prompt_country, &country, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	cv := oauth2.GenerateVerifier()
	oc := psa.Oauth2Config(strings.ToLower(vehicleConf.Type), strings.ToLower(country))

	// challenge := oauth2.S256ChallengeFromVerifier(cv)

	// redirect_uri := fmt.Sprintf("%s://oauth2redirect/%s", td.scheme, country)
	// authorize_url := fmt.Sprintf("https://idpcvs.%s/am/oauth2/authorize?%s", td.realm, url.Values{
	// 	"client_id":             []string{td.clientid},
	// 	"redirect_uri":          []string{redirect_uri},
	// 	"response_type":         []string{"code"},
	// 	"scope":                 []string{"openid profile"},
	// 	"code_challenge":        []string{challenge},
	// 	"code_challenge_method": []string{"S256"},
	// }.Encode())

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

	// 	tok, err := oc.Exchange(context.Background(), code, oauth2.VerifierOption(cv))

	headers := map[string]string{
		"Authorization": transport.BasicAuthHeader(oc.ClientID, oc.ClientSecret),
		"Content-type":  request.FormContent,
	}
	data := url.Values{
		"grant_type":    []string{"authorization_code"},
		"code":          []string{code},
		"code_verifier": []string{cv},
		"redirect_uri":  []string{oc.RedirectURL},
	}

	req, _ := request.New(http.MethodPost, oc.Endpoint.TokenURL, strings.NewReader(data.Encode()), headers)

	log := util.NewLogger("psa")
	client := request.NewHelper(log)
	var res oauth.Token
	if err := client.DoJSON(req, &res); err != nil {
		return nil, err
	}

	tok := (*oauth2.Token)(&res)

	return tok, nil
}
