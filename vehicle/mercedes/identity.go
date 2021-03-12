package mercedes

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type Identity struct {
}

func AuthConfig(id, secret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  "https://id.mercedes-benz.com/as/token.oauth2",
			AuthURL:   "https://id.mercedes-benz.com/as/authorization.oauth2",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		// Scopes: []string{"mb:vehicle:status:general", "mb:user:pool:reader", "offline_access"},
		Scopes: []string{"offline_access"},
	}
}

func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// openURL opens the specified URL in the default browser of the user.
func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func GenerateTokens(oc *oauth2.Config) error {
	state := state()
	url := oc.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "login consent"))
	fmt.Println(url)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	http.HandleFunc("/", v.redirectHandler(ctx, wg, state))
	go func() {
		wg.Done()
		http.ListenAndServe(":34972", nil)
	}()

	wg.Add(1)
	if err := openURL(url); err != nil {
		return err
	}

	go func() {
		time.Sleep(10 * time.Second)
		wg.Done()
	}()

	wg.Wait()

	return nil
}
