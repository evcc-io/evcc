package fritz

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"golang.org/x/text/encoding/unicode"
)

// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID_english_2021-05-03.pdf
const SessionTimeout = 15 * time.Minute

// CreateChallengeResponse creates the Fritzbox challenge response string
func CreateChallengeResponse(challenge, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		return "", err
	}

	hash := md5.Sum([]byte(utf16le))
	md5hash := hex.EncodeToString(hash[:])

	return challenge + "-" + md5hash, nil
}
