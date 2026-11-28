package legacy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginAction(t *testing.T) {
	action, err := loginAction(`"url": "\/realms/polestar", "loginAction": "https:\/\/auth.polestar.com\/authenticate?session_code=a&tab_id=b",`)
	require.NoError(t, err)
	assert.Equal(t, "https://auth.polestar.com/authenticate?session_code=a&tab_id=b", action)

	_, err = loginAction(`url: "/as/resume"`)
	assert.Error(t, err)

	_, err = loginAction(`"loginAction": "https://auth.polestar.com.example.org/authenticate"`)
	assert.Error(t, err)
}

func TestLoginError(t *testing.T) {
	assert.ErrorContains(t, loginError(`"pageId": "login", "summary": "Invalid username or password."`), "invalid username or password")
	assert.ErrorContains(t, loginError(`"pageId": "login-otp"`), "sign in once at polestar.com")
	assert.ErrorContains(t, loginError(`"pageId": "login"`), "missing authorization code")
}
