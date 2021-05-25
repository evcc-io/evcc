package tasks

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

const Sma TaskType = "shm"

func init() {
	registry.Add(Sma, SMAHandlerFactory)
}

type ShmResult struct {
	Serial string
	Http   bool
}

func SMAHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := SMAHandler{
		Timeout: 5 * time.Second,
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type SMAHandler struct {
	mux      sync.Mutex
	listener *sma.Listener
	Timeout  time.Duration
}

func (h *SMAHandler) httpAvailable(ip string) bool {
	uri := fmt.Sprintf("https://%s", ip)

	client := http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(uri)
	if err != nil {
		return false
	}

	resp.Body.Close()
	return true
}

func (h *SMAHandler) Test(log *util.Logger, in ResultDetails) (res []ResultDetails) {
	h.mux.Lock()

	if h.listener != nil {
		h.mux.Unlock()
		return nil
	}

	var err error
	if h.listener, err = sma.New(log); err != nil {
		log.ERROR.Println("shm:", err)
		return nil
	}
	h.mux.Unlock()

	resC := make(chan sma.Telegram)
	h.listener.Subscribe(sma.Any, resC)

	timer := time.NewTimer(h.Timeout)
WAIT:
	for {
		select {
		case t := <-resC:
			// eliminate duplicates
			for _, r := range res {
				if r.ShmResult != nil && r.ShmResult.Serial == t.Serial {
					continue WAIT
				}
			}

			out := ResultDetails{
				IP: t.Addr,
				ShmResult: &ShmResult{
					Serial: t.Serial,
					Http:   h.httpAvailable(t.Addr),
				},
			}

			res = append(res, out)

		case <-timer.C:
			break WAIT
		}
	}

	return res
}
