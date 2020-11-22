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
		Timeout: 5 * timeout,
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type KEBAHandler struct {
	mux      sync.Mutex
	listener *keba.Listener
	Timeout  time.Duration
}

func (h *KEBAHandler) Test(log *util.Logger, ip string) []interface{} {
	h.mux.Lock()

	if h.listener == nil {
		var err error
		if h.listener, err = keba.New(log); err != nil {
			log.ERROR.Println("keba:", err)
		}

		h.mux.Unlock()
		return nil
	}

	h.mux.Unlock()

	resC := make(chan keba.UDPMsg)
	h.listener.Subscribe(ip, resC)

	sender, err := keba.NewSender(ip)
	if err != nil {
		log.ERROR.Println("keba:", err)
		return nil
	}

	timer := time.NewTimer(h.Timeout)
WAIT:
	for {
		go func() {
			_ = sender.Send("report 1")
		}()

		select {
		case t := <-resC:
			log.INFO.Println(t)
			if t.Report == nil {
				continue
			}

			r := KebaResult{
				Addr:   t.Addr,
				Serial: t.Report.Serial,
			}

			return []interface{}{r}

		case <-timer.C:
			break WAIT
		}
	}

	return nil
}
