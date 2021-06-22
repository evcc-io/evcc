package tariff

import (
	"sync"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/tariff/awattar"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

type Awattar struct {
	mux   sync.Mutex
	Cheap float64
	data  []awattar.PriceInfo
}

var _ api.Tariff = (*Awattar)(nil)

func NewAwattar(other map[string]interface{}) (*Awattar, error) {
	cc := Awattar{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	go cc.Run()

	return &cc, nil
}

func (t *Awattar) Run() {
	log := util.NewLogger("awattar")
	client := request.NewHelper(log)

	for ; true; <-time.NewTicker(time.Hour).C {
		var res awattar.Prices
		if err := client.GetJSON(awattar.URI, &res); err != nil {
			log.ERROR.Println(err)
			continue
		}

		t.mux.Lock()
		t.data = res.Data
		t.mux.Unlock()
	}
}

func (t *Awattar) IsCheap() bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	for i := len(t.data) - 1; i >= 0; i-- {
		pi := t.data[i]

		if pi.StartTimestamp.Before(time.Now()) && pi.EndTimestamp.After(time.Now()) {
			return pi.Marketprice <= t.Cheap
		}
	}

	return false
}
