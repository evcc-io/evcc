package aiways

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Identity struct {
	*request.Helper
	user, hash, token string
}

type TokenProvider interface {
	Token() string
}

// NewIdentity creates BMW identity
func NewIdentity(log *util.Logger) *Identity {
	v := &Identity{
		Helper: request.NewHelper(log),
	}

	return v
}

func (v *Identity) Login(user, password string) (int64, error) {
	hash := md5.New()
	hash.Write([]byte(password))

	v.user = user
	v.hash = hex.EncodeToString(hash.Sum(nil))

	var res User

	data := struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{
		Account:  v.user,
		Password: v.hash,
	}

	uri := fmt.Sprintf("%s/aiways-passport-service/passport/login/password", URI)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	if err == nil {
		if err = v.DoJSON(req, &res); err == nil && res.Data == nil {
			err = errors.New(res.Message)
		}

		if err == nil {
			v.token = res.Data.Token
			return res.Data.UserID, nil
		}
	}

	return 0, err
}

func (v *Identity) Token() string {
	return v.token
}
