package ocpp

import "time"

type Transaction struct {
	ID              int       `yaml:"id"`
	IDTag           string    `yaml:"idTag"`
	Start           time.Time `yaml:"startTimestamp"`
	End             time.Time `yaml:"endTimestamp,omitempty"`
	MeterValueStart int64     `yaml:"meterValueStart"`
	MeterValueStop  int64     `yaml:"meterValueStop"`
	Charged         int64     `yaml:"charged"`
}

func NewTransaction(id int, idTag string, start time.Time, meterValue int) Transaction {
	return Transaction{
		ID:              id,
		IDTag:           idTag,
		Start:           start,
		MeterValueStart: int64(meterValue),
	}
}

func (t *Transaction) Finish(idTag string, end time.Time, meterValue int) {
	t.IDTag = idTag
	t.MeterValueStop = int64(meterValue)
	t.End = end
}
