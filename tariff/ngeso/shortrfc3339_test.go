package ngeso

import (
	"testing"
)

var (
	tTsBytes      = []byte("2023-04-20T14:30Z")
	tTsStringRepr = "2023-04-20 14:30:00 +0000 UTC"
)

func TestMarshalling(t *testing.T) {
	// Firstly, test that we can unmarshal into a struct.
	ct := shortRFC3339Timestamp{}
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
	if string(res) != string(tTsBytes) {
		t.Errorf("time did not marshal successfully: got %s, expected %s", res, tTsBytes)
	}
}
