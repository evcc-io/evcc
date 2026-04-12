package fritz

// FritzDECT settings
type Settings struct {
	URI, AIN, User, Password string
	Legacy                   bool // use legacy homeautoswitch.lua API
}
