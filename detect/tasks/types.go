package tasks

import (
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/copier"
)

type ResultDetails struct {
	IP           string
	Port         int           `json:",omitempty"`
	Topic        string        `json:",omitempty"`
	ModbusResult *ModbusResult `json:",omitempty"`
	KebaResult   *KebaResult   `json:",omitempty"`
	SmaResult    *SmaResult    `json:",omitempty"`
}

func (d *ResultDetails) Clone() ResultDetails {
	var c ResultDetails
	if err := copier.Copy(&c, *d); err != nil {
		panic(err)
	}
	return c
}

type Result struct {
	Task
	ResultDetails
	Attributes map[string]interface{} // TODO remove, only used for post-processing
}

type TaskType string

type Task struct {
	ID      string
	Type    TaskType
	Depends string
	Config  map[string]interface{}
	TaskHandler
}

type TaskHandler interface {
	Test(log *util.Logger, in ResultDetails) []ResultDetails
}
