package vw

import (
	"errors"
	"io"

	"github.com/PuerkitoBio/goquery"
)

// FormVars holds HTML form input values required for login
type FormVars struct {
	Action     string
	Csrf       string
	RelayState string
	Hmac       string
}

// FormValues extracts FormVars from given HTML document
func FormValues(reader io.Reader, id string) (map[string]string, error) {
	vars := make(map[string]string)

	doc, err := goquery.NewDocumentFromReader(reader)
	if err == nil {
		// only interested in meta tag?
		if meta := doc.Find("meta[name=_csrf]"); id == "meta" {
			if meta.Length() != 1 {
				return vars, errors.New("unexpected length")
			}

			csrf, exists := meta.Attr("content")
			if !exists {
				return vars, errors.New("meta not found")
			}
			vars["_csrf"] = csrf
			return vars, nil
		}

		form := doc.Find(id).First()
		if form.Length() != 1 {
			return vars, errors.New("unexpected length")
		}

		action, exists := form.Attr("action")
		if !exists {
			return vars, errors.New("attribute not found")
		}
		vars["action"] = action

		form.Find("input").Each(func(_ int, el *goquery.Selection) {
			if name, ok := el.Attr("name"); ok {
				vars[name], _ = el.Attr("value")
			}
		})
	}

	return vars, err
}
