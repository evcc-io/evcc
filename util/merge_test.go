package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestMerge(t *testing.T) {
	oc := oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: "tu",
		},
	}

	res, err := Merge(oc, oauth2.Config{ClientID: "cid"})
	require.NoError(t, err)
	require.Equal(t, "cid", res.ClientID)
}
