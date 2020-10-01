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
func FormValues(reader io.Reader, id string) (FormVars, error) {
	vars := FormVars{}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err == nil {
		// only interested in meta tag?
		if meta := doc.Find("meta[name=_csrf]"); id == "meta" {
			if meta.Length() != 1 {
				return vars, errors.New("unexpected length")
			}

			var exists bool
			vars.Csrf, exists = meta.Attr("content")
			if !exists {
				return vars, errors.New("meta not found")
			}
			return vars, nil
		}

		form := doc.Find(id).First()
		if form.Length() != 1 {
			return vars, errors.New("unexpected length")
		}

		var exists bool
		vars.Action, exists = form.Attr("action")
		if !exists {
			return vars, errors.New("attribute not found")
		}

		vars.Csrf, err = attr(form, "input[name=_csrf]", "value")
		if err == nil {
			vars.RelayState, err = attr(form, "input[name=relayState]", "value")
		}
		if err == nil {
			vars.Hmac, err = attr(form, "input[name=hmac]", "value")
		}
	}

	return vars, err
}

func attr(doc *goquery.Selection, path, attr string) (res string, err error) {
	sel := doc.Find(path)
	if sel.Length() != 1 {
		return "", errors.New("unexpected length")
	}

	v, exists := sel.Attr(attr)
	if !exists {
		return "", errors.New("attribute not found")
	}

	return v, nil
}
