package zero

type UnitNumberAnswer struct {
	Unitnumber string //"123456",
	Name       string
}

type ErrorAnswer struct {
	Error string
}

type ZeroState struct {
	Unitnumber       string  //"123456",
	Name             string  //"538ZFAZ76LCK00000",
	Unittype         string  //"5",
	Unitmodel        string  //"6",
	Mileage          float64 `json:",string"` //"4382.46",
	Software_version string  //"190430",
	Logic_state      string  //"2"
	Reason           string  //"2",
	Response         string  //"0",
	Driver           string  //"0",
	Latitude         float64 `json:",string"` //":"51.5000",
	Longitude        float64 `json:",string"` //"4.5000",
	Altitude         string  //:"0",
	Gps_valid        string  //:"0",
	Gps_connected    string  //:"1",
	Satellites       string  //"0",
	Velocity         string  //"1",
	Heading          string  //"344",
	Emergency        string  //:"0",
	Shock            string  //:"",
	Ignition         string  //:"0",
	Door             string  //:"0",
	Hood             string  //:"0",
	Volume           string  //:"0",
	Water_temp       string  //:"",
	Oil_pressure     string  //:"0",
	Main_voltage     float64 //:13.08,
	Analog1          float64 //":"0.09",
	Analog2          float64 //":"0.09",
	Analog3          float64 //":"0.09",
	Siren            string  //:"0",
	Lock             string  //:"0",
	Int_lights       string  //:"0",
	Datetime_utc     string  //:"20191030162309",
	Datetime_actual  string  //:"20191102113548"
	Address          string  //:"YourCity, YourStreet",
	Perimeter        string  //:"",
	Color            int32   //:2,
	Soc              int32   //:91,
	Tipover          int32   //:0,
	Charging         int32   //:1,
	Chargecomplete   int32   // 0,
	Pluggedin        int32   //:1,
	Chargingtimeleft int32   //:0
	Storage          int32
	Battery          int32
}

type UnitData []UnitNumberAnswer
