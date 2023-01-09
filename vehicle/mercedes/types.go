package mercedes

type EVResponse struct {
	Soc struct {
		Value     int64 `json:",string"`
		Timestamp int64
	}
	RangeElectric struct {
		Value     int64 `json:",string"`
		Timestamp int64
	}
}
