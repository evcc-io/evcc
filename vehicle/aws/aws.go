package aws

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

type Credentials struct {
	Value  credentials.Value
	Expiry time.Time
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
			Value: credentials.Value{
				AccessKeyID:     n.AccessKeyID,
				SecretAccessKey: n.SecretKey,
				SessionToken:    n.SessionToken,
			},
			Expiry: time.Unix(int64(n.Expiration), 0),
		}
	}

	return err
}

type CredentialsProvider struct {
	creds Credentials
}

func (p *CredentialsProvider) Retrieve() (credentials.Value, error) {
	return p.creds.Value, nil
}

func (p *CredentialsProvider) IsExpired() bool {
	return p.creds.Expiry.After(time.Now().Add(-time.Minute))
}

func NewEphemeralCredentials(creds Credentials) *credentials.Credentials {
	return credentials.NewCredentials(&CredentialsProvider{
		creds: creds,
	})
}
