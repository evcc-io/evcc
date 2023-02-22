package daheimladen

type (
	ChargePointStatus  string
	RemoteStartStatus  string
	RemoteStopStatus   string
	ConfigKey          string
	ChangeConfigStatus string
)

const BASE_URL string = "https://api.daheimladen.com/v1"

const (
	AVAILABLE ChargePointStatus = "Available"
	PREPARING ChargePointStatus = "Preparing"
	CHARGING  ChargePointStatus = "Charging"
	FINISHING ChargePointStatus = "Finishing"
	FAULTED   ChargePointStatus = "Faulted"

	REMOTE_START_ACCEPTED RemoteStartStatus = "Accepted"
	REMOTE_START_REJECTED RemoteStartStatus = "Rejected"

	REMOTE_STOP_ACCEPTED RemoteStopStatus = "Accepted"
	REMOTE_STOP_REJECTED RemoteStopStatus = "Rejected"

	CHANGE_CONFIG_ACCEPTED ChangeConfigStatus = "Accepted"
	CHANGE_CONFIG_REJECTED ChangeConfigStatus = "Rejected"

	CHARGE_RATE ConfigKey = "ChargeRate"

	EVCC_IDTAG string = "evcc"
)
