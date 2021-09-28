package tasks

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/charger/keba"
	"github.com/evcc-io/evcc/util"
)

const Keba TaskType = "keba"

func init() {
	registry.Add(Keba, KEBAHandlerFactory)
}

type KebaResult struct {
	Addr, Serial string
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

func (h *KEBAHandler) Test(log *util.Logger, in ResultDetails) []ResultDetails {
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
	h.listener.Subscribe(in.IP, resC)

	sender, err := keba.NewSender(log, in.IP)
	if err != nil {
		log.ERROR.Println("keba:", err)
		return nil
	}

	timer := time.NewTimer(h.Timeout)

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

			out := in.Clone()
			out.KebaResult = &KebaResult{
				Addr:   t.Addr,
				Serial: t.Report.Serial,
			}

			return []ResultDetails{out}

		case <-timer.C:
			return nil
		}
	}
}
