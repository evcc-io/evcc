package bmw

type (
	Region struct {
		AuthURI, CocoApiURI string
		Token
		Authenticate
	}
	Token struct {
		Authorization string
	}
	Authenticate struct {
		ClientID, State string
	}
)

var regions = map[string]Region{
	"NA": {
		"https://login.bmwusa.com/gcdm",
		"https://cocoapi.bmwgroup.us",
		Token{
			Authorization: "Basic NTQzOTRhNGItYjZjMS00NWZlLWI3YjItOGZkM2FhOTI1M2FhOmQ5MmYzMWMwLWY1NzktNDRmNS1hNzdkLTk2NmY4ZjAwZTM1MQ==",
		},
		Authenticate{
			ClientID: "54394a4b-b6c1-45fe-b7b2-8fd3aa9253aa",
			State:    "rgastJbZsMtup49-Lp0FMQ",
		},
	},
	"EU": {
		"https://customer.bmwgroup.com/gcdm",
		"https://cocoapi.bmwgroup.com",
		Token{
			Authorization: "Basic MzFjMzU3YTAtN2ExZC00NTkwLWFhOTktMzNiOTcyNDRkMDQ4OmMwZTMzOTNkLTcwYTItNGY2Zi05ZDNjLTg1MzBhZjY0ZDU1Mg==",
		},
		Authenticate{
			ClientID: "31c357a0-7a1d-4590-aa99-33b97244d048",
			State:    "cEG9eLAIi6Nv-aaCAniziE_B6FPoobva3qr5gukilYw",
		},
	},
}
