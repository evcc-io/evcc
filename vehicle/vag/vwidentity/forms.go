package vwidentity

import (
	"encoding/json"
	"errors"
	"fmt"
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
	var res CredentialParams

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, r); err != nil {
		return res, err
	}

	re := regexp.MustCompile(`(?s)window._IDK\s*=\s*(.*?)[;<]`)
	match := re.FindAllStringSubmatch(buf.String(), -1)

	tmpl := strings.ReplaceAll(match[0][1], `'`, `"`)
	for _, v := range []string{"templateModel", "currentLocale", "csrf_parameterName", "csrf_token", "userSession", "userId", "countryOfResidence"} {
		tmpl = strings.Replace(tmpl, v, fmt.Sprintf(`"%s"`, v), 1)
	}

	err := json.Unmarshal([]byte(tmpl), &res)

	return res, err
}
