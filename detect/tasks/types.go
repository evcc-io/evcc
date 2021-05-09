package tasks

import (
	"fmt"

	"github.com/andig/evcc/util"
	"github.com/jinzhu/copier"
)

type Details struct {
	IP   string
	Port int
	// Attributes   map[string]interface{}
	Topic        string
	ModbusResult *ModbusResult
	KebaResult   *KebaResult
	SmaResult    *SmaResult
}

func (d *Details) Clone() Details {
	fmt.Println(d)
	var c Details
	copier.Copy(&c, *d)
	fmt.Println(c)
	return c
}

// func (d *Details) Attr(key string, val interface{}) {
// 	if d.Attributes == nil {
// 		d.Attributes = make(map[string]interface{})
// 	}
// 	d.Attributes[key] = val
// }

type Result struct {
	Task
	Details    Details
	Attributes map[string]interface{}
}

type Task struct {
	ID, Type string
	Depends  string
	Config   map[string]interface{}
	TaskHandler
}

type TaskHandler interface {
	Test(log *util.Logger, in Details) []Details
}
