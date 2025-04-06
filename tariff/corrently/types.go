package corrently

type Forecast struct {
	Support       string `json:"support"`
	License       string `json:"license"`
	Info          string `json:"info"`
	Documentation string `json:"documentation"`
	Commercial    string `json:"commercial"`
	Signee        string `json:"signee"`
	Forecast      []struct {
		Epochtime     int     `json:"epochtime"`
		Eevalue       int     `json:"eevalue"`
		Ewind         int     `json:"ewind"`
		Esolar        int     `json:"esolar"`
		Ensolar       int     `json:"ensolar"`
		Enwind        int     `json:"enwind"`
		Sci           int     `json:"sci"`
		Gsi           float64 `json:"gsi"`
		TimeStamp     int64   `json:"timeStamp"`
		Energyprice   string  `json:"energyprice"`
		Co2GStandard  int     `json:"co2_g_standard"`
		Co2GOekostrom int     `json:"co2_g_oekostrom"`
		Timeframe     struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"timeframe"`
		Iat       int64  `json:"iat"`
		Zip       string `json:"zip"`
		Signature string `json:"signature"`
	} `json:"forecast"`
	Location struct {
		Zip       string `json:"zip"`
		City      string `json:"city"`
		Signature string `json:"signature"`
	} `json:"location"`
	Err     bool
	Message any
}
