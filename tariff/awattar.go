package tariff

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"math/rand"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/awattar"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Awattar struct {
	mux         sync.Mutex
	log         *util.Logger
	uri         string
	cheap       float64
	cheapactive bool
	last        time.Time
	data        []awattar.PriceInfo
}

type MyByPrices awattar.ByPrices

var _ api.Tariff = (*Awattar)(nil)

func fakeAwattar() awattar.Prices {
	start := time.Now()
	start = start.Add(-2 * time.Minute)

	f := new(awattar.Prices)

	for i := 0; i < 20; i++ {
		p := new(awattar.PriceInfo)
		p.StartTimestamp = start
		start = start.Add(1 * time.Minute)
		p.EndTimestamp = start
		p.Marketprice = float64(100 + rand.Intn(40))
		p.Unit = "Eur/MWh"
		f.Data = append(f.Data, *p)
	}

	return *f
}

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
		log:   util.NewLogger("awattar"),
		cheap: cc.Cheap,
		uri:   fmt.Sprintf(awattar.RegionURI, strings.ToLower(cc.Region)),
	}

	//t.data = fakeAwattar().Data
	go t.Run()

	return t, nil
}

// MyByPrices implement sort.interface based on Marketprice field
func (a MyByPrices) Len() int {
	return len(a)
}

func (a MyByPrices) Less(i, j int) bool {
	if a[i].Marketprice == a[j].Marketprice {
		return a[i].StartTimestamp.After(a[j].StartTimestamp)
	}
	return a[i].Marketprice < a[j].Marketprice
}

func (a MyByPrices) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (t *Awattar) Run() {
	client := request.NewHelper(t.log)

	for ; true; <-time.NewTicker(time.Hour).C {
		var res awattar.Prices
		if true {
			if err := client.GetJSON(t.uri, &res); err != nil {
				t.log.ERROR.Println(err)
				continue
			}

			t.last = res.Data[len(res.Data)-1].EndTimestamp
			sort.Sort(MyByPrices(res.Data))

			t.mux.Lock()
			t.data = res.Data
			t.mux.Unlock()
		} else {
			t.last = t.data[len(t.data)-1].EndTimestamp
			sort.Sort(MyByPrices(t.data))
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
			return pi.Marketprice / 1e3, nil // convert EUR/MWh to EUR/KWh
		}
	}

	return 0, errors.New("unable to find current awattar price")
}

func (t *Awattar) IsCheap() (bool, error) {
	price, err := t.CurrentPrice()
	return price <= t.cheap, err
}

func (t *Awattar) Rates() ([]api.Rate, error) {
	var res []api.Rate

	for _, r := range t.data {
		ar := api.Rate{
			Start: r.StartTimestamp,
			End:   r.EndTimestamp,
			Price: r.Marketprice / 1e3,
		}
		res = append(res, ar)
	}

	return res, nil
}
