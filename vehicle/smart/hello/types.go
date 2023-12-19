package hello

import (
	"strconv"
	"strings"
)

const ResponseOK = 1000

type ResponseCode int

func (rc *ResponseCode) UnmarshalJSON(data []byte) error {
	i, err := strconv.Atoi(strings.Trim(string(data), `"`))
	if err == nil {
		*rc = ResponseCode(i)
	}
	return err
}

type AppToken struct {
	ExpiresIn    int
	AccessToken  string
	UserId       string
	RefreshToken string
}

type Vehicle struct {
	VIN string
}

type StatusResponse struct{}
