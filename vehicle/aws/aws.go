package aws

import (
	"encoding/json"
	"time"
)

type Credentials struct {
	AccessKeyID  string
	SecretKey    string
	SessionToken string
	Expiry       time.Time
}

func (c *Credentials) UnmarshalJSON(data []byte) error {
	var n struct {
		AccessKeyID  string
		Expiration   float64
		SecretKey    string
		SessionToken string
	}

	err := json.Unmarshal(data, &n)
	if err == nil {
		*c = Credentials{
			AccessKeyID:  n.AccessKeyID,
			SecretKey:    n.SecretKey,
			SessionToken: n.SessionToken,
			Expiry:       time.Unix(int64(n.Expiration), 0),
		}
	}

	return err
}
