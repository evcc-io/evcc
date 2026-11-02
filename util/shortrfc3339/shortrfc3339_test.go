package shortrfc3339

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	tTsBytes      = []byte("2023-04-20T14:30Z")
	tTsStringRepr = "2023-04-20 14:30:00 +0000 UTC"
)

func TestMarshalling(t *testing.T) {
	// Firstly, test that we can unmarshal into a struct.
	ct := Timestamp{}
	if err := ct.UnmarshalJSON(tTsBytes); err != nil {
		t.Fatal(err)
	}
	if ct.Time.String() != tTsStringRepr {
		t.Errorf("time did not unmarshal successfully: got %s, expected %s", ct.Time.String(), tTsStringRepr)
	}

	// Now test remarshalling.
	res, err := ct.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != `"`+string(tTsBytes)+`"` {
		t.Errorf("time did not marshal successfully: got %s, expected \"%s\"", res, tTsBytes)
	}
}

func TestMarshalJSONIsValidJSON(t *testing.T) {
	v := struct {
		From *Timestamp `json:"from"`
	}{&Timestamp{time.Date(2023, 4, 20, 14, 30, 0, 0, time.UTC)}}

	res, err := json.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `{"from":"2023-04-20T14:30Z"}`, string(res))

	var back struct {
		From Timestamp `json:"from"`
	}
	require.NoError(t, json.Unmarshal(res, &back))
	assert.True(t, back.From.Equal(v.From.Time), "round trip: got %s, expected %s", back.From.Time, v.From.Time)
}

func TestMarshalJSONConvertsToUTC(t *testing.T) {
	ct := Timestamp{time.Date(2023, 4, 20, 16, 30, 0, 0, time.FixedZone("CEST", 2*3600))}

	res, err := ct.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"2023-04-20T14:30Z"`, string(res))
}

func TestMarshalJSONValue(t *testing.T) {
	ts := Timestamp{time.Date(2023, 4, 20, 16, 30, 0, 0, time.FixedZone("CEST", 2*3600))}

	// a top-level value and a value field are not addressable
	res, err := json.Marshal(ts)
	require.NoError(t, err)
	assert.Equal(t, `"2023-04-20T14:30Z"`, string(res))

	res, err = json.Marshal(struct {
		From Timestamp `json:"from"`
	}{ts})
	require.NoError(t, err)
	assert.Equal(t, `{"from":"2023-04-20T14:30Z"}`, string(res))

	res, err = json.Marshal(Timestamp{})
	require.NoError(t, err)
	assert.Equal(t, `null`, string(res))
}
