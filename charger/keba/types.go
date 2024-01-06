package keba

// RFID contains access credentials
type RFID struct {
	Tag string
}

// Report contains report id and device serial
type Report struct {
	ID     int    `json:"ID,string"`
	Serial string `json:"Serial"`
}

// Report1 is the report 1 command answer
type Report1 struct {
	ID        int    `json:"ID,string"`
	Serial    string `json:"Serial"`
	Product   string `json:"Product"`
	Firmware  string `json:"Firmware"`
	COMModule int    `json:"COM-module"`
	Sec       int64  `json:"Sec"`
}

// Report2 is the report 2 command answer
type Report2 struct {
	ID             int    `json:"ID,string"`
	Serial         string `json:"Serial"`
	State          int    `json:"State"`
	Error1         int    `json:"Error1"`
	Error2         int    `json:"Error2"`
	Plug           int    `json:"Plug"`
	AuthON         int    `json:"AuthON"`
	AuthReq        int    `json:"Authreq"`
	EnableSys      int    `json:"Enable sys"`
	EnableUser     int    `json:"Enable user"`
	MaxCurr        int    `json:"Max curr"`
	MaxCurrPercent int    `json:"Max curr %"`
	CurrHW         int    `json:"Curr HW"`
	Curruser       int    `json:"Curr user"`
	CurrFS         int    `json:"Curr FS"`
	TmoFS          int    `json:"Tmo FS"`
	CurrTimer      int    `json:"Curr timer"`
	TmoCT          int    `json:"Tmo CT"`
	SetEnergy      int    `json:"Setenergy"`
	Output         int    `json:"Output"`
	Input          int    `json:"Input"`
	Sec            int64  `json:"Sec"`
}

// Report3 is the report 3 command answer
type Report3 struct {
	ID     int    `json:"ID,string"`
	Serial string `json:"Serial"`
	U1     int64  `json:"U1"`
	U2     int64  `json:"U2"`
	U3     int64  `json:"U3"`
	I1     int64  `json:"I1"`
	I2     int64  `json:"I2"`
	I3     int64  `json:"I3"`
	P      int64  `json:"P"`
	PF     int64  `json:"PF"`
	EPres  int64  `json:"E pres"`
	ETotal int64  `json:"E total"`
	Sec    int64  `json:"Sec"`
}

// Report100 is the report 100 command answer
type Report100 struct {
	ID        int    `json:"ID,string"`
	Serial    string `json:"Serial"`
	SessionID int64  `json:"SessionID"`
	CurrHW    int    `json:"Curr HW"`
	EStart    int64  `json:"E start"`
	EPres     int64  `json:"E pres"`
	Started   string `json:"started"`
	Ended     string `json:"ended"`
	StartedS  int64  `json:"started[s]"`
	EndedS    int64  `json:"ended[s]"`
	Reason    int    `json:"reason"`
	TimeQ     int    `json:"timeQ"`
	RFIDTag   string `json:"RFID tag"`
	RFIDClass string `json:"RFID class"`
	Sec       int64  `json:"Sec"`
}
