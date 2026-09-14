package auth

import (
	"context"

	"golang.org/x/oauth2"
)

// WithExchangeOptions adds parameters to the authorization code exchange
func WithExchangeOptions(opts ...oauth2.AuthCodeOption) func(o *OAuth) {
	return func(o *OAuth) {
		o.exchangeOpts = opts
	}
}

// WithLoginHook runs before the login url is generated, e.g. to register with the provider
func WithLoginHook(hook func(context.Context) error) func(o *OAuth) {
	return func(o *OAuth) {
		o.loginHook = hook
	}
}

func WithOauthDeviceFlowOption() func(o *OAuth) {
	return func(o *OAuth) {
		o.deviceFlow = true
	}
}

func WithTokenStorerOption(ts func(*oauth2.Token) any) func(o *OAuth) {
	return func(o *OAuth) {
		o.tokenStorer = ts
	}
}

func WithTokenRetrieverOption(tr func(string, *oauth2.Token) error) func(o *OAuth) {
	return func(o *OAuth) {
		o.tokenRetriever = tr
	}
}
