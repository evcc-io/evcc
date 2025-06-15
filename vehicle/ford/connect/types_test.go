package connect

import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	data := []byte(`{
		"timestamp": "05-24-2024 15:58:56"
	}`)

	var res TimedValue
	require.NoError(t, json.Unmarshal(data, &res, json.MatchCaseInsensitiveNames(true)))

	require.Equal(t, "2024-05-24T15:58:56Z", res.Timestamp.Format(time.RFC3339))
}
