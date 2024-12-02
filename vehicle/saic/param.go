package saic

type (
	Region struct {
		ApiURL string
	}
)

var regions = map[string]Region{
	"EU": {
		"https://gateway-mg-eu.soimt.com/api.app/v1/",
	},
	"AU": {
		"https://gateway-mg-au.soimt.com/api.app/v1/",
	},
}
