package vag

import (
	"net/url"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
)

func ChallengeAndVerifier(q url.Values) func(url.Values) {
	cvc, _ := cv.CreateCodeVerifier()

	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", cvc.CodeChallengeS256())

	return func(q url.Values) {
		q.Set("code_verifier", cvc.CodeChallengePlain())
	}
}
