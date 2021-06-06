package bluelink

import (
	"bufio"
	_ "embed" // embedded files
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

//go:embed 693a33fa-c117-43f2-ae3b-61a02d24f417
var kia string

//go:embed 99cfff84-f4e2-4be8-a5ed-e5b755eb6581
var hyundai string

// Stamps collects stamps for a single brand
type Stamps map[string][]string

var stamps = Stamps{
	"693a33fa-c117-43f2-ae3b-61a02d24f417": unpack(kia),
	"99cfff84-f4e2-4be8-a5ed-e5b755eb6581": unpack(hyundai),
}

var (
	mu     sync.Mutex
	client *request.Helper
)

var brands = map[string]string{
	"693a33fa-c117-43f2-ae3b-61a02d24f417": "kia",
	"99cfff84-f4e2-4be8-a5ed-e5b755eb6581": "hyundai",
}

func download(id, brand string) {
	var res []string
	uri := fmt.Sprintf("https://raw.githubusercontent.com/neoPix/bluelinky-stamps/master/%s.json", brand)
	if err := client.GetJSON(uri, &res); err == nil {
		if len(res) < 100 {
			return
		}

		mu.Lock()
		stamps[id] = res[:99]
		mu.Unlock()
	}
}

// Downloader updates stamps according to https://github.com/Hacksore/bluelinky/pull/144
func Downloader() {
	if client == nil {
		return
	}

	client = request.NewHelper(util.NewLogger("http"))

	for ; true; <-time.NewTicker(24 * time.Hour).C {
		for k, v := range brands {
			download(k, v)
		}
	}
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
func (s Stamps) New(id string) string {
	mu.Lock()
	defer mu.Unlock()

	source, ok := s[id]
	if !ok {
		panic(id)
	}
	return source[rand.Intn(len(source))]
}
