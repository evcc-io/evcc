package push

import (
	"github.com/SherClockHolmes/webpush-go"
	"github.com/evcc-io/evcc/server/db/settings"
)

const (
	keyVAPIDPrivate = "push.vapid.private"
	keyVAPIDPublic  = "push.vapid.public"
)

// VAPIDKeys returns the cached VAPID key pair, generating one on first call.
func VAPIDKeys() (private, public string, err error) {
	private, err = settings.String(keyVAPIDPrivate)
	if err == nil {
		public, err = settings.String(keyVAPIDPublic)
		if err == nil {
			return private, public, nil
		}
	}

	// Generate new VAPID key pair.
	private, public, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}

	settings.SetString(keyVAPIDPrivate, private)
	settings.SetString(keyVAPIDPublic, public)

	return private, public, nil
}
