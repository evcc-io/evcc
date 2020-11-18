package detect

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/andig/evcc/meter/sma"
	"github.com/andig/evcc/util"
)

type SmaResult struct {
	Addr, Serial string
	Http         bool
}

func init() {
	registry.Add("sma", SMAHandlerFactory)
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
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// 	return http.ErrUseLastResponse
		// },
	}

	resp, err := client.Get(uri)
	if err != nil {
		fmt.Println(err)
		return false
	}

	fmt.Println("ok")

	resp.Body.Close()
	return true
}

func (h *SMAHandler) Test(log *util.Logger, ip string) (res []interface{}) {
	h.mux.Lock()

	if h.listener != nil {
		h.mux.Unlock()
		return nil
	}

	var err error
	if h.listener, err = sma.New(log); err != nil {
		log.ERROR.Println("sma:", err)
		return nil
	}
	h.mux.Unlock()

	resC := make(chan sma.Telegram)
	h.listener.Subscribe(sma.All, resC)

	timer := time.NewTimer(h.Timeout)
WAIT:
	for {
		select {
		case t := <-resC:
			// eliminate duplicates
			for _, r := range res {
				if r.(SmaResult).Serial == t.Serial {
					continue WAIT
				}
			}

			r := SmaResult{
				Addr:   t.Addr,
				Serial: t.Serial,
				Http:   h.httpAvailable(t.Addr),
			}

			res = append(res, r)

		case <-timer.C:
			break WAIT
		}
	}

	fmt.Println(res)
	return res
}
