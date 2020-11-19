package detect

import (
	"sync"
	"time"

	"github.com/andig/evcc/charger/keba"
	"github.com/andig/evcc/util"
)

type KebaResult struct {
	Addr, Serial string
}

func init() {
	registry.Add("keba", KEBAHandlerFactory)
}

func KEBAHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := KEBAHandler{
		Timeout: 5 * time.Second,
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type KEBAHandler struct {
	mux      sync.Mutex
	listener *keba.Listener
	Timeout  time.Duration
}

func (h *KEBAHandler) Test(log *util.Logger, ip string) (res []interface{}) {
	h.mux.Lock()

	if h.listener != nil {
		h.mux.Unlock()
		return nil
	}

	var err error
	if h.listener, err = keba.New(log); err != nil {
		log.ERROR.Println("keba:", err)
		return nil
	}
	h.mux.Unlock()

	resC := make(chan keba.UDPMsg)
	h.listener.Subscribe(keba.Any, resC)

	timer := time.NewTimer(h.Timeout)
WAIT:
	for {
		select {
		case t := <-resC:
			if t.Report == nil {
				continue
			}

			// eliminate duplicates
			for _, r := range res {
				if r.(KebaResult).Serial == t.Report.Serial {
					continue WAIT
				}
			}

			r := KebaResult{
				Addr:   t.Addr,
				Serial: t.Report.Serial,
			}

			res = append(res, r)

		case <-timer.C:
			break WAIT
		}
	}

	return res
}
