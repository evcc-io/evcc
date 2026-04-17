package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/server/push"
	"github.com/evcc-io/evcc/util"
)

// pushCheckHandler returns 200 if the given endpoint is registered, 404 if not.
func pushCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" {
			jsonError(w, http.StatusBadRequest, errors.New("endpoint required"))
			return
		}
		ok, err := push.SubscriptionExists(endpoint)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		jsonWrite(w, struct{}{})
	}
}

// pushVapidKeyHandler returns the VAPID public key for the frontend to subscribe.
func pushVapidKeyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, publicKey, err := push.VAPIDKeys()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}
		jsonWrite(w, map[string]string{"publicKey": publicKey})
	}
}

// pushUnsubscribeHandler removes a single browser push subscription by endpoint.
func pushUnsubscribeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" {
			jsonError(w, http.StatusBadRequest, errors.New("endpoint required"))
			return
		}
		if err := push.DeleteSubscription(endpoint); err != nil {
			util.NewLogger("push").ERROR.Printf("delete subscription: %v", err)
		}
		jsonWrite(w, struct{}{})
	}
}

// pushSubscribeHandler stores a Web Push subscription.
func pushSubscribeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sub push.Subscription

		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if sub.Endpoint == "" || sub.Auth == "" || sub.P256dh == "" {
			jsonError(w, http.StatusBadRequest, errors.New("endpoint, auth and p256dh are required"))
			return
		}

		if err := push.SaveSubscription(sub); err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonWrite(w, struct{}{})
	}
}
