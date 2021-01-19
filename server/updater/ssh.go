// +build gokrazy

package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const service = "/user/breakglass"

var (
	client *http.Client
	xsrf   string
)

func (u *watch) sshHandler(w http.ResponseWriter, r *http.Request) {
	var status, target struct {
		Active bool `json:"active"`
	}

	// GET ssh status
	var err error
	status.Active, err = u.sshStatus()

	if err == nil {
		w.Header().Add("Content-type", "application/json")
		err = json.NewEncoder(w).Encode(&status)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if r.Method == http.MethodGet {
		return
	} else if r.Method != http.MethodPost {
		http.Error(w, "method forbidden", http.StatusBadRequest)
		return
	}

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// start/stop service
	if status.Active != target.Active {
		if err := u.sshEnable(target.Active); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (u *watch) sshStatus() (bool, error) {
	var status struct{ Stopped bool }

	uri := fmt.Sprintf("http://gokrazy:%s@%s:%d/status?path=%s", Password, Host, Port, url.QueryEscape(service))
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err == nil {
		req.Header.Add("Content-type", "application/json")

		var resp *http.Response
		resp, err = client.Do(req)

		if err == nil {
			defer resp.Body.Close()
			err = json.NewDecoder(resp.Body).Decode(&status)

			if err == nil {
				for _, c := range resp.Cookies() {
					if c.Name == "gokrazy_xsrf" {
						xsrf = c.Value
						break
					}
				}

				if xsrf == "" {
					err = errors.New("missing xsrf token")
				}
			}
		}
	}

	return !status.Stopped, err
}

func (u *watch) sshEnable(enable bool) error {
	data := url.Values{
		"xsrftoken": []string{xsrf},
		"path":      []string{service},
	}

	action := "stop"
	if enable {
		action = "restart"
	}

	uri := fmt.Sprintf("http://gokrazy:%s@%s:%d/%s", Password, Host, Port, action)
	_, err := client.PostForm(uri, data)

	return err
}
