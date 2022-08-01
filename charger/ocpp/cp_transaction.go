package ocpp

import "time"

type Transaction struct {
	ID    int       `yaml:"id"`
	IDTag string    `yaml:"idTag"`
	Start time.Time `yaml:"startTimestamp"`
	End   time.Time `yaml:"endTimestamp,omitempty"`
}

func NewTransaction(id int, idTag string, start time.Time) Transaction {
	return Transaction{
		ID:    id,
		IDTag: idTag,
		Start: start,
	}
}

func (t *Transaction) Finish(idTag string, end time.Time) {
	t.IDTag = idTag
	t.End = end
}
