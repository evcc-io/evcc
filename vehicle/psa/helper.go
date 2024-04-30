package psa

import (
	"fmt"
	"strings"
)

var (
	identities = make(map[string]*Identity)
)

func getInstance(brand, user string) *Identity {
	return identities[fmt.Sprintf("%s.%s", strings.ToLower(brand), strings.ToLower(user))]
}

func addInstance(brand, user string, identity *Identity) {
	identities[fmt.Sprintf("%s.%s", strings.ToLower(brand), strings.ToLower(user))] = identity
}

func SettingsKey(brand, user string) string {
	return fmt.Sprintf("psa.%s.%s", strings.ToLower(brand), strings.ToLower(user))
}
