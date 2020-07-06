package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/andig/evcc/util"
	influxdb2 "github.com/influxdata/influxdb-client-go"
)

// InfluxConfig is the influx db configuration
type InfluxConfig struct {
	URL      string
	Database string
	Token    string
	Org      string
	User     string
	Password string
	Interval time.Duration
}

// Influx is a influx publisher
type Influx struct {
	sync.Mutex
	log      *util.Logger
	client   influxdb2.Client
	org      string
	database string
}

// NewInfluxClient creates new publisher for influx
func NewInfluxClient(url, token, org, user, password, database string) *Influx {
	log := util.NewLogger("iflx")

	// InfluxDB v1 compatibility
	if token == "" && user != "" {
		token = fmt.Sprintf("%s:%s", user, password)
	}

	client := influxdb2.NewClient(url, token)

	return &Influx{
		log:      log,
		client:   client,
		org:      org,
		database: database,
	}
}

// Run Influx publisher
func (m *Influx) Run(in <-chan util.Param) {
	writer := m.client.WriteApi(m.org, m.database)

	// log errors
	go func() {
		for err := range writer.Errors() {
			m.log.ERROR.Println(err)
		}
	}()

	// add points to batch for async writing
	for param := range in {
		// allow nil value to be written as series gaps
		if _, ok := param.Val.(float64); param.Val != nil && !ok {
			continue
		}

		tags := map[string]string{}
		if param.LoadPoint != "" {
			tags["loadpoint"] = param.LoadPoint
		}

		fields := map[string]interface{}{
			"value": param.Val,
		}

		// write asynchronously
		m.log.TRACE.Printf("write %s=%v (%v)", param.Key, param.Val, tags)
		p := influxdb2.NewPoint(param.Key, tags, fields, time.Now())
		writer.WritePoint(p)
	}

	m.client.Close()
}
