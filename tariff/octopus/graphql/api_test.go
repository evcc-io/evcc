package graphql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOctopusGraphQLAccountFiltration(t *testing.T) {
	validAccountNumber := "A-AAAAAAAA"
	var noAccounts []krakenAccount
	oneAccount := []krakenAccount{
		{Number: validAccountNumber},
	}
	multipleAccounts := []krakenAccount{
		{Number: validAccountNumber},
		{Number: "X-XXXXXXXX"},
		{Number: "Y-YYYYYYYY"},
		{Number: "Z-ZZZZZZZZ"},
	}

	var accNum string

	// No accounts (invalid state)
	_, err := filterAccount(noAccounts, "")
	require.ErrorIs(t, err, ErrNoAccounts)

	// One account, no filtration
	accNum, err = filterAccount(oneAccount, "")
	require.NoError(t, err)
	require.Equal(t, accNum, validAccountNumber)

	// One account, valid filtration
	accNum, err = filterAccount(oneAccount, validAccountNumber)
	require.NoError(t, err)
	require.Equal(t, accNum, validAccountNumber)

	// One account, invalid filtration (invalid state)
	_, err = filterAccount(oneAccount, "0-00000000")
	require.ErrorIs(t, err, ErrAccountNotFound)

	// Multiple accounts, no filtration (invalid state)
	_, err = filterAccount(multipleAccounts, "")
	require.ErrorIs(t, err, ErrMultipleAccounts)

	// Multiple accounts, valid filtration
	accNum, err = filterAccount(multipleAccounts, validAccountNumber)
	require.NoError(t, err)
	require.Equal(t, accNum, validAccountNumber)

	// Multiple accounts, invalid filtration (invalid state)
	_, err = filterAccount(multipleAccounts, "0-00000000")
	require.ErrorIs(t, err, ErrAccountNotFound)
}
