package main

// do not commit this file, it's only for testing!

import (
	"fmt"

	"github.com/evcc-io/evcc/charger/viessmann"
	"github.com/evcc-io/evcc/util"
)

func main() {
	// util.LogLevel("trace", nil) // Enable trace logging
	log := util.NewLogger("viessmann").Redact(viessmann.Password)
	token_source, _ := viessmann.NewIdentity(log, viessmann.User, viessmann.Password)
	token, _ := token_source.Token()
	fmt.Println(token)
	fmt.Println(token.Expiry)

	// TODO test if we can refresh the token - the below doesn't work:
	// cannot use token_source (variable of interface type oauth2.TokenSource) as oauth.TokenRefresher value in argument to
	// oauth.RefreshTokenSource: oauth2.TokenSource does not implement oauth.TokenRefresher (missing method RefreshToken)
	// refresh_token := oauth.RefreshTokenSource(token, token_source)
	// fmt.Println(refresh_token)
}
