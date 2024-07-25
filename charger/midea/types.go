package midea

const (
	BaseUrl = "https://mapp.appsmb.com"
	AppKey  = "3742e9e5842d4ad59c2db887e12449f9"
)

type (
	LoginId struct {
		LoginId string
	}

	Login struct {
		OriginPrivateVersion string
		Nickname             string
		SessionId            string
		AccessToken          string
		UserId               string
		VersionCode          string
		LeftCount            string
	}

	Homegroup struct {
		Id       string
		Nickname string
	}
	HomegroupList struct {
		List []Homegroup
	}

	Appliance struct {
		MasterId     string
		Des          string
		ActiveStatus int `json:",string"`
		OnlineStatus int `json:",string"`
		Name         string
		ModelNumber  string
		Id           string
		Sn           string
		Type         string
		Tsn          string
		Mac          string
	}

	ApplianceList struct {
		List []Appliance
	}
)
