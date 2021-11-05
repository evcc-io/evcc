package bluelink

import (
	"bufio"
	_ "embed" // embedded files
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

//go:embed 693a33fa-c117-43f2-ae3b-61a02d24f417
var kia string

//go:embed 014d2225-8495-4735-812d-2616334fd15d
var hyundai string

// StampsRegistry collects stamps for a single brand
type StampsRegistry map[string][]string

const (
	KiaAppID     = "693a33fa-c117-43f2-ae3b-61a02d24f417"
	HyundaiAppID = "014d2225-8495-4735-812d-2616334fd15d"
)

var Stamps = StampsRegistry{
	KiaAppID:     unpack(kia),
	HyundaiAppID: unpack(hyundai),
}

var (
	mu      sync.Mutex
	updater map[string]struct{} = make(map[string]struct{})

	client = request.NewHelper(util.NewLogger("http"))
	brands = map[string]string{
		KiaAppID:     "kia",
		HyundaiAppID: "hyundai",
	}
)

func download(log *util.Logger, id, brand string) {
	var res []string
	uri := fmt.Sprintf("https://raw.githubusercontent.com/neoPix/bluelinky-stamps/master/%s.json", brand)

	err := client.GetJSON(uri, &res)
	if err != nil {
		log.ERROR.Println(err)
		return
	}

	if len(res) < 100 {
		return
	}

	mu.Lock()
	Stamps[id] = res
	mu.Unlock()
}

// updateStamps updates stamps according to https://github.com/Hacksore/bluelinky/pull/144
func updateStamps(log *util.Logger, id string) {
	if _, ok := updater[id]; ok {
		return
	}

	updater[id] = struct{}{}
	download(log, id, brands[id])

	go func() {
		for range time.NewTicker(24 * time.Hour).C {
			download(log, id, brands[id])
		}
	}()
}

func unpack(source string) (res []string) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return
}

// New creates a new stamp
func (s StampsRegistry) New(id string) string {
	mu.Lock()
	defer mu.Unlock()

	source, ok := s[id]
	if !ok {
		panic(id)
	}
	return source[rand.Intn(len(source))]
}
