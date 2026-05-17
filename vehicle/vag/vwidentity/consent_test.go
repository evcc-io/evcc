package vwidentity

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarketingConsentCallback(t *testing.T) {
	// non-consent page (normal final redirect) is ignored
	plain, err := url.Parse("https://identity.vwgroup.io/oidc/v1/oauth/sso?clientId=foo")
	require.NoError(t, err)
	cb, err := marketingConsentCallback(plain)
	require.NoError(t, err)
	require.Nil(t, cb)

	// marketing consent page yields the embedded OIDC callback (issue #29760)
	consent, err := url.Parse("https://identity.vwgroup.io/signin-service/v1/consent/marketing/user-id/foo@apps_vw-dilab_com/0?relayState=rs&hmac=h&callback=" +
		url.QueryEscape("https://identity.vwgroup.io/oidc/v1/oauth/client/callback/success?user_id=user-id&client_id=foo@apps_vw-dilab_com&scopes=openid profile mbb&consentedScopes=openid profile mbb&relayState=rs&hmac=cbhmac"))
	require.NoError(t, err)

	cb, err = marketingConsentCallback(consent)
	require.NoError(t, err)
	require.NotNil(t, cb)
	require.Equal(t, "/oidc/v1/oauth/client/callback/success", cb.Path)
	// spaces in query values are normalized so the request is valid
	require.Equal(t, "openid profile mbb", cb.Query().Get("consentedScopes"))
	require.NotContains(t, cb.RawQuery, " ")

	// marketing consent page without a callback is an explicit error
	noCallback, err := url.Parse("https://identity.vwgroup.io/signin-service/v1/consent/marketing/user-id/foo/0?relayState=rs")
	require.NoError(t, err)
	_, err = marketingConsentCallback(noCallback)
	require.Error(t, err)
}
