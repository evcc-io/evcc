package mercedes

type EVResponse struct {
	SoC struct {
		Value     int64 `json:",string"`
		Timestamp int64
	}
	RangeElectric struct {
		Value     int64 `json:",string"`
		Timestamp int64
	}
}
