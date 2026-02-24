package vwidentity

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FormVars holds HTML form input values required for login
type FormVars struct {
	Action string
	Inputs map[string]string
}

// FormValues extracts FormVars from given HTML document
func FormValues(reader io.Reader, id string) (FormVars, error) {
	vars := FormVars{Inputs: make(map[string]string)}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err == nil {
		// only interested in meta tag?
		if meta := doc.Find("meta[name=_csrf]"); id == "meta" {
			if meta.Length() != 1 {
				return vars, errors.New("meta not found")
			}

			csrf, exists := meta.Attr("content")
			if !exists {
				return vars, errors.New("meta attribute not found")
			}
			vars.Inputs["_csrf"] = csrf
			return vars, nil
		}

		form := doc.Find(id).First()
		if form.Length() != 1 {
			return vars, errors.New("form not found")
		}

		action, exists := form.Attr("action")
		if !exists {
			return vars, errors.New("form attribute not found")
		}
		vars.Action = action

		form.Find("input").Each(func(_ int, el *goquery.Selection) {
			if name, ok := el.Attr("name"); ok {
				vars.Inputs[name], _ = el.Attr("value")
			}
		})
	}

	return vars, err
}

type CredentialParams struct {
	TemplateModel struct {
		Hmac          string `json:"hmac"`
		RelayState    string `json:"relayState"`
		PostAction    string `json:"postAction"`
		IdentifierUrl string `json:"identifierUrl"`
		Error         string `json:"error"`
	} `json:"templateModel"`
	CurrentLocale     string `json:"currentLocale"`
	CsrfParameterName string `json:"csrf_parameterName"`
	CsrfToken         string `json:"csrf_token"`
}

func ParseCredentialsPage(r io.ReadCloser) (CredentialParams, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return CredentialParams{}, err
	}

	return parseCredentials(string(b))
}

func parseCredentials(body string) (CredentialParams, error) {
	var res CredentialParams

	// find js block
	match := regexp.MustCompile(`(?s)window._IDK\s*=\s*(.*?)[;<]`).FindAllStringSubmatch(body, -1)
	if len(match) < 1 || len(match[0]) < 2 {
		return res, errors.New("IDK form not found")
	}

	// clean quotes
	quotes1 := strings.ReplaceAll(match[0][1], `'`, `"`)
	quotes2 := regexp.MustCompile(`\s(\w+)(?s)*:`).ReplaceAllString(quotes1, ` "$1":`)

	// strip , }
	tmpl := regexp.MustCompile(`(?s),\s+}`).ReplaceAllString(quotes2, "}")

	err := json.Unmarshal([]byte(tmpl), &res)

	return res, err
}
