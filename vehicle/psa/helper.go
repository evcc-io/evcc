package psa

import (
	"fmt"
	"strings"
)

var (
	identities = make(map[string]*Identity)
)

func getInstance(brand, account string) *Identity {
	return identities[fmt.Sprintf("%s.%s", strings.ToLower(brand), strings.ToLower(account))]
}

func addInstance(brand, account string, identity *Identity) {
	identities[fmt.Sprintf("%s.%s", strings.ToLower(brand), strings.ToLower(account))] = identity
}

func SettingsKey(brand, account string) string {
	return fmt.Sprintf("psa.%s.%s", strings.ToLower(brand), strings.ToLower(account))
}
