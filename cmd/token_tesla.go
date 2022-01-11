package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/util/request"
	"github.com/manifoldco/promptui"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"
)

// copied from https://github.com/bogosj/tesla
func getUsernameAndPassword() (string, string, error) {
	user, err := (&promptui.Prompt{
		Label:   "Username",
		Pointer: promptui.PipeCursor,
		Validate: func(s string) error {
			if len(s) == 0 {
				return errors.New("len(s) == 0")
			}
			return nil
		},
	}).Run()
	if err != nil {
		return "", "", err
	}

	password, err := (&promptui.Prompt{
		Label:   "Password",
		Mask:    '*',
		Pointer: promptui.PipeCursor,
		Validate: func(s string) error {
			if len(s) == 0 {
				return errors.New("len(s) == 0")
			}
			return nil
		},
	}).Run()
	if err != nil {
		return "", "", err
	}

	return user, password, nil
}

func codePrompt(ctx context.Context, devices []tesla.Device) (tesla.Device, string, error) {
	var i int
	if len(devices) > 1 {
		var err error
		i, _, err = (&promptui.Select{
			Label:   "Device",
			Items:   devices,
			Pointer: promptui.PipeCursor,
		}).Run()
		if err != nil {
			return tesla.Device{}, "", fmt.Errorf("select device: %w", err)
		}
	}

	code, err := (&promptui.Prompt{
		Label:   "Passcode",
		Pointer: promptui.PipeCursor,
		Validate: func(s string) error {
			if len(s) != 6 {
				return errors.New("len(s) != 6")
			}
			return nil
		},
	}).Run()
	if err != nil {
		return tesla.Device{}, "", err
	}

	return devices[i], strings.TrimSpace(code), nil
}

func captchaPrompt(ctx context.Context, svg io.Reader) (string, error) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "evcc-*.svg")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, svg); err != nil {
		return "", fmt.Errorf("cannot write temp file: %w", err)
	}

	_ = tmpFile.Close()

	if err := open.Run(tmpFile.Name()); err != nil {
		return "", fmt.Errorf("cannot open captcha for display: %w", err)
	}

	fmt.Println("Captcha is now being opened in default application for svg files.")

	captcha, err := (&promptui.Prompt{
		Label:   "Captcha",
		Pointer: promptui.PipeCursor,
		Validate: func(s string) error {
			if len(s) < 4 {
				return errors.New("len(s) < 4")
			}
			return nil
		},
	}).Run()

	return strings.TrimSpace(captcha), err
}

func teslaToken() (*oauth2.Token, error) {
	username, password, err := getUsernameAndPassword()
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewHelper(log).Client)
	client, err := tesla.NewClient(
		ctx,
		tesla.WithMFAHandler(codePrompt),
		tesla.WithCaptchaHandler(captchaPrompt),
		tesla.WithCredentials(username, password),
	)
	if err != nil {
		return nil, err
	}

	token, err := client.Token()
	if err != nil {
		return nil, err
	}

	return token, nil
}
