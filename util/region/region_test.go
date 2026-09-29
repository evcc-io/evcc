package region

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/currency"
)

func TestParse(t *testing.T) {
	tests := []struct {
		code    string
		wantErr bool
	}{
		{"DE", false},
		{"US", false},
		{"de", true},  // wrong case
		{"XX", true},  // not a country
		{"", true},    // empty
		{"DEU", true}, // alpha-3 not supported
	}

	for _, tc := range tests {
		_, err := Parse(tc.code)
		if tc.wantErr {
			assert.Error(t, err, tc.code)
		} else {
			assert.NoError(t, err, tc.code)
		}
	}
}

func TestCurrency(t *testing.T) {
	tests := []struct {
		code string
		want currency.Unit
		ok   bool
	}{
		{"DE", currency.EUR, true},
		{"AT", currency.EUR, true},
		{"US", currency.USD, true},
		{"GB", currency.GBP, true},
		{"CH", currency.CHF, true},
		{"XX", currency.Unit{}, false},
		{"", currency.Unit{}, false},
	}

	for _, tc := range tests {
		got, ok := Currency(tc.code)
		assert.Equal(t, tc.ok, ok, tc.code)
		if tc.ok {
			assert.Equal(t, tc.want, got, tc.code)
		}
	}
}
