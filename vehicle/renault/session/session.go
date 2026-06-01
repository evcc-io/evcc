package session

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/evcc-io/evcc/server/db/settings"
)

const verificationCodeMessage = "Enter the verification code from your Renault email."

// Subject returns the stable provider auth id for a Renault account.
func Subject(user, region string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(user)) + ":" + strings.ToLower(strings.TrimSpace(region))))
	return "renault-" + hex.EncodeToString(h[:])[:12]
}

// VerificationCodeMessage returns the UI prompt for Renault TFA.
func VerificationCodeMessage() string {
	return verificationCodeMessage
}

// SettingsKey returns the persisted trusted device key for a Renault account.
func SettingsKey(user, region string) string {
	return Subject(user, region) + "-gmid"
}

// StoredGMID returns the trusted Gigya member id for a Renault account.
func StoredGMID(user, region string) string {
	gmid, _ := settings.String(SettingsKey(user, region))
	return gmid
}

// StoreGMID stores the trusted Gigya member id for a Renault account.
func StoreGMID(user, region, gmid string) {
	settings.SetString(SettingsKey(user, region), gmid)
}

// DeleteGMID removes the trusted Gigya member id for a Renault account.
func DeleteGMID(user, region string) error {
	return settings.Delete(SettingsKey(user, region))
}
