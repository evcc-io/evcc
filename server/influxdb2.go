package server

import (
	"sync"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
	influxdb2 "github.com/influxdata/influxdb-client-go"
)

// Influx2 is a influx publisher
type Influx2 struct {
	sync.Mutex
	log    *util.Logger
	client influxdb2.InfluxDBClient
	writer influxdb2.WriteApi
}

// NewInflux2Client creates new publisher for influx
func NewInflux2Client(url, token, org, bucket string) *Influx2 {
	log := util.NewLogger("iflx")

	client := influxdb2.NewClient(url, token)
	writer := client.WriteApi(org, bucket)

	return &Influx2{
		log:    log,
		client: client,
		writer: writer,
	}
}

// Run Influx publisher
func (m *Influx2) Run(in <-chan core.Param) {
	// log errors
	go func() {
		for err := range m.writer.Errors() {
			m.log.ERROR.Println(err)
		}
	}()

	// add points to batch for async writing
	for param := range in {
		// allow nil value to be written as series gaps
		if _, ok := param.Val.(float64); param.Val != nil && !ok {
			continue
		}

		p := influxdb2.NewPoint(
			param.Key,
			map[string]string{
				"loadpoint": param.LoadPoint,
			},
			map[string]interface{}{
				"value": param.Val,
			},
			time.Now(),
		)

		// write asynchronously
		m.writer.WritePoint(p)
	}

	m.client.Close()
}
