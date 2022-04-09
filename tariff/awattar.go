package tariff

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/awattar"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/log"
	"github.com/evcc-io/evcc/util/request"
)

type Awattar struct {
	mux   sync.Mutex
	log   log.Logger
	uri   string
	cheap float64
	data  []awattar.PriceInfo
}

var _ api.Tariff = (*Awattar)(nil)

func NewAwattar(other map[string]interface{}) (*Awattar, error) {
	cc := struct {
		Cheap  float64
		Region string
	}{
		Region: "DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &Awattar{
		log:   log.NewLogger("awattar"),
		cheap: cc.Cheap,
		uri:   fmt.Sprintf(awattar.RegionURI, strings.ToLower(cc.Region)),
	}

	go t.Run()

	return t, nil
}

func (t *Awattar) Run() {
	client := request.NewHelper(t.log)

	for ; true; <-time.NewTicker(time.Hour).C {
		var res awattar.Prices
		if err := client.GetJSON(t.uri, &res); err != nil {
			t.log.Error(err)
			continue
		}

		t.mux.Lock()
		t.data = res.Data
		t.mux.Unlock()
	}
}

func (t *Awattar) CurrentPrice() (float64, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	for i := len(t.data) - 1; i >= 0; i-- {
		pi := t.data[i]

		if pi.StartTimestamp.Before(time.Now()) && pi.EndTimestamp.After(time.Now()) {
			return pi.Marketprice / 1000, nil // convert EUR/MWh to EUR/KWh
		}
	}

	return 0, errors.New("unable to find current awattar price")
}

func (t *Awattar) IsCheap() (bool, error) {
	price, err := t.CurrentPrice()
	return price <= t.cheap, err
}
