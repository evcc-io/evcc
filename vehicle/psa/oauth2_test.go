package psa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVauxhallDefaultsToGB(t *testing.T) {
	assert.Equal(t, "mymvxsdk://oauth2redirect/gb", Oauth2Config("vauxhall", "").RedirectURL)
	assert.Equal(t, "mymvxsdk://oauth2redirect/gb", Oauth2Config("vauxhall", "gb").RedirectURL)
}
