package vag

import (
	"net/url"

	"golang.org/x/oauth2"
)

func ChallengeAndVerifier(q url.Values) func(url.Values) {
	cv := oauth2.GenerateVerifier()

	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", oauth2.S256ChallengeFromVerifier(cv))

	return func(q url.Values) {
		q.Set("code_verifier", cv)
	}
}
