package tasks

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"gitlab.com/bboehmke/sunny"
)

const Sma TaskType = "sma"

func init() {
	registry.Add(Sma, SMAHandlerFactory)
}

type SmaResult struct {
	Serial string
	Http   bool
}

func SMAHandlerFactory(conf map[string]interface{}) (TaskHandler, error) {
	handler := SMAHandler{
		Timeout:  5 * time.Second,
		Password: "0000",
	}

	err := util.DecodeOther(conf, &handler)

	return &handler, err
}

type SMAHandler struct {
	mux      sync.Mutex
	handled  bool
	Timeout  time.Duration
	Password string
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

	if h.handled {
		h.mux.Unlock()
		return nil
	}

	connection, err := sunny.NewConnection("")
	if err != nil {
		log.ERROR.Println("sma:", err)
		return nil
	}

	devices := connection.SimpleDiscoverDevices(h.Password)
	h.handled = true
	h.mux.Unlock()

	for _, device := range devices {
		res = append(res, ResultDetails{
			IP: device.Address().IP.String(),
			SmaResult: &SmaResult{
				Serial: strconv.FormatInt(int64(device.SerialNumber()), 10),
				Http:   h.httpAvailable(device.Address().IP.String()),
			},
		})
	}

	return res
}
