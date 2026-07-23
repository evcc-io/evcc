package auth

import "golang.org/x/oauth2"

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
