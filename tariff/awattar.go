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

func fake_awattar() awattar.Prices {
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
		//fmt.Printf("Length %d\n", len(f.Data))
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

	//t.data = fake_awattar().Data
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

func (t *Awattar) IsCheap(duration time.Duration, end time.Time) bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	if end.Before(time.Now()) {
		t.cheapactive = false
		return false
	}

	duration = time.Duration(float64(duration) * 1.05) // increase by 5%

	// Save same duration until next price info update
	if end.After(t.last) {
		duration_old := duration
		duration = time.Duration(float64(duration) * float64(time.Until(t.last)) / float64(time.Until(end)))
		t.log.DEBUG.Printf("reduced duration from %s to %s until got new priceinfo after %s\n", duration_old.Round(time.Minute), duration.Round(time.Minute), t.last.Round(time.Minute))
	}

	t.log.DEBUG.Printf("charge duration: %s, end: %v, find best prices:\n", duration.Round(time.Minute), end.Round(time.Second))

	var pi awattar.PriceInfo
	var sum time.Duration
	cheap_slot := false
	cnt_expected_slots := 0
	cur_slot_nr := 0

	for i := 0; i < len(t.data); i++ {
		pi = t.data[i]

		if pi.StartTimestamp.Before(time.Now()) && pi.EndTimestamp.After(time.Now()) {
			return pi.Marketprice / 1000, nil // convert EUR/MWh to EUR/KWh

		if pi.EndTimestamp.Before(time.Now()) { // old data
			continue
		}

		if !(pi.StartTimestamp.Before(end)) { // charge ends before
			continue
		}

		// timeslot already startet
		pstart := pi.StartTimestamp
		if pstart.Before(time.Now()) {
			pstart = time.Now()
		}

		// timeslot ends after charge finish time
		pend := pi.EndTimestamp
		if pend.After(end) {
			pend = end
		}

		cnt_expected_slots++
		delta := pend.Sub(pstart)
		sum += delta
		t.log.TRACE.Printf("  Slot from: %v to %v price %f, timesum %s\n", pi.StartTimestamp.Round(time.Second), pi.EndTimestamp.Round(time.Second), pi.Marketprice, sum)

		// current timeslot is a cheap one
		if pi.StartTimestamp.Before(time.Now()) && pi.EndTimestamp.After(time.Now()) && duration > 0 {
			cheap_slot = true // rename to cheapSlotNow
			cur_slot_nr = i
		}

		// we found all necessary cheap slots to charge to targetSoC
		if sum > duration {
			break
		}
	}

	if cheap_slot {
		// use the most expensive slot as little as possible, but do not disable on last charging slot
		if cur_slot_nr == cnt_expected_slots-1 && !(t.cheapactive && cnt_expected_slots == 1) {
			if sum <= duration {
				t.log.DEBUG.Printf("cheap timeslot, charging...\n")
				t.cheapactive = true
			} else {
				if t.cheapactive && sum > duration+10*time.Minute {
					t.log.DEBUG.Printf("cheap timeslot, delayed start for %s\n", (sum - duration).Round(time.Minute))
					t.cheapactive = false
				}
			}
		} else {
			t.cheapactive = true
		}
	} else {
		t.cheapactive = false
	}

	if t.cheapactive {
		return true
	}

	cheap := pi.Marketprice/10 <= t.cheap // Eur/MWh conversion
	if cheap {
		t.log.DEBUG.Printf("low marketprice, charging")
	}

	return cheap
}
