package messenger

import (
	"encoding/json"

	webpushlib "github.com/SherClockHolmes/webpush-go"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/push"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("webpush", NewPushSenderFromConfig)
}

// NewPushSenderFromConfig creates a new Web Push sender from config. No params required.
func NewPushSenderFromConfig(_ map[string]any) (api.Messenger, error) {
	return NewPushSender(), nil
}

// PushSender implements api.Messenger using the Web Push protocol.
// It sends push notifications to all registered browser subscriptions.
type PushSender struct {
	log *util.Logger
}

// NewPushSender creates a new Web Push sender.
func NewPushSender() *PushSender {
	return &PushSender{log: util.NewLogger("webpush")}
}

// Send implements api.Messenger. It sends a push notification to all subscriptions.
func (s *PushSender) Send(title, msg string) {
	subs, err := push.AllSubscriptions()
	if err != nil {
		s.log.ERROR.Printf("load subscriptions: %v", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	privateKey, publicKey, err := push.VAPIDKeys()
	if err != nil {
		s.log.ERROR.Printf("vapid keys: %v", err)
		return
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"body":  msg,
	})
	if err != nil {
		s.log.ERROR.Printf("marshal payload: %v", err)
		return
	}

	for _, sub := range subs {
		wsSub := &webpushlib.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpushlib.Keys{
				Auth:   sub.Auth,
				P256dh: sub.P256dh,
			},
		}

		resp, err := webpushlib.SendNotification(payload, wsSub, &webpushlib.Options{
			VAPIDPrivateKey: privateKey,
			VAPIDPublicKey:  publicKey,
			Subscriber:      "mailto:evcc@localhost",
			TTL:             86400,
		})
		if err != nil {
			s.log.ERROR.Printf("send to %s: %v", sub.Endpoint, err)
			continue
		}
		resp.Body.Close()
		// Remove expired or invalid subscriptions (410 Gone, 404 Not Found).
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			if delErr := push.DeleteSubscription(sub.Endpoint); delErr != nil {
				s.log.ERROR.Printf("delete stale subscription: %v", delErr)
			}
		}
	}
}
