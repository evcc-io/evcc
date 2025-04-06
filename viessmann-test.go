package main

// do not commit this file, it's only for testing!

import (
	"fmt"
	"github.com/evcc-io/evcc/charger/viessmann"
	"github.com/evcc-io/evcc/util"
	// "golang.org/x/oauth2"
)

func main() {
  // fmt.Println(viessmann.User)
  fmt.Println(viessmann.OAuth2Config)

	util.LogLevel("trace", nil) // Enable trace logging
	log := util.NewLogger("viessmann").Redact(viessmann.Password)
	identity, err := viessmann.NewIdentity(log, viessmann.User, viessmann.Password)
	if err != nil {
      fmt.Println(err)
	// 	return nil, fmt.Errorf("login failed: %w", err)
	}
  fmt.Println(identity)

	// fmt.Println(viessmann.login2(identity))


  // like func login()
	// cv := oauth2.GenerateVerifier()

	// state := lo.RandomString(16, lo.AlphanumericCharset)
	// uri := OAuth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(cv),
	// 	oauth2.SetAuthURLParam("audience", ApiURI),
	// 	oauth2.SetAuthURLParam("ui_locales", "de-DE"),
	// )
  // fmt.Println(user, password, cv)

}


