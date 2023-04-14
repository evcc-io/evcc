package elering

const URI = "https://dashboard.elering.ee/api"

type NpsPrice struct {
	Success bool
	Data    map[string][]Price
}

type Price struct {
	Timestamp int64
	Price     float64
}
