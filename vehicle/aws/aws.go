package aws

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
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

type CredentialsProvider struct {
	Value  credentials.Value
	expiry time.Time
}

func (p *CredentialsProvider) Retrieve() (credentials.Value, error) {
	return p.Value, nil
}

func (p *CredentialsProvider) IsExpired() bool {
	return p.expiry.After(time.Now().Add(-time.Minute))
}

func NewEphemeralCredentials(c Credentials) *credentials.Credentials {
	return credentials.NewCredentials(&CredentialsProvider{
		Value: credentials.Value{
			AccessKeyID:     c.AccessKeyID,
			SecretAccessKey: c.SecretKey,
			SessionToken:    c.SessionToken,
		},
		expiry: c.Expiry,
	})
}
