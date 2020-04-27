package server

import (
	"sync"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
	influxdb "github.com/influxdata/influxdb1-client/v2"
)

const (
	influxWriteTimeout  = 10 * time.Second
	influxWriteInterval = 30 * time.Second
	precision           = "s"
)

// Influx is a influx publisher
type Influx struct {
	sync.Mutex
	log        *util.Logger
	client     influxdb.Client
	points     []*influxdb.Point
	pointsConf influxdb.BatchPointsConfig
	interval   time.Duration
}

// NewInfluxClient creates new publisher for influx
func NewInfluxClient(
	url string,
	database string,
	interval time.Duration,
	user string,
	password string,
) *Influx {
	log := util.NewLogger("iflx")

	if database == "" {
		log.FATAL.Fatal("missing database")
	}
	if interval == 0 {
		interval = influxWriteInterval
	}

	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     url,
		Username: user,
		Password: password,
		Timeout:  influxWriteTimeout,
	})
	if err != nil {
		log.FATAL.Fatalf("error creating client: %v", err)
	}

	// check connection
	go func(client influxdb.Client) {
		if _, _, err := client.Ping(influxWriteTimeout); err != nil {
			log.FATAL.Fatalf("%v", err)
		}
	}(client)

	return &Influx{
		log:      log,
		client:   client,
		interval: interval,
		pointsConf: influxdb.BatchPointsConfig{
			Database:  database,
			Precision: precision,
		},
	}
}

// writeBatchPoints asynchronously writes the collected points
func (m *Influx) writeBatchPoints() {
	m.Lock()

	// get current batch
	if len(m.points) == 0 {
		m.Unlock()
		return
	}

	// create new batch
	batch, err := influxdb.NewBatchPoints(m.pointsConf)
	if err != nil {
		m.log.ERROR.Print(err)
		m.Unlock()
		return
	}

	// replace current batch
	points := m.points
	m.points = nil
	m.Unlock()

	// write batch
	batch.AddPoints(points)
	m.log.TRACE.Printf("writing %d point(s)", len(points))

	if err := m.client.Write(batch); err != nil {
		m.log.ERROR.Print(err)

		// put points back at beginning of next batch
		m.Lock()
		m.points = append(points, m.points...)
		m.Unlock()
	}
}

// asyncWriter periodically calls writeBatchPoints
func (m *Influx) asyncWriter(exit <-chan struct{}) <-chan struct{} {
	done := make(chan struct{}) // signal writer stopped

	// async batch writer
	go func() {
		ticker := time.NewTicker(m.interval)
		for {
			select {
			case <-ticker.C:
				m.writeBatchPoints()
			case <-exit:
				ticker.Stop()
				m.writeBatchPoints()
				close(done)
				return
			}
		}
	}()

	return done
}

// Run Influx publisher
func (m *Influx) Run(in <-chan core.Param) {
	exit := make(chan struct{}) // exit signals to stop writer
	done := m.asyncWriter(exit) // done signals writer stopped

	// add points to batch for async writing
	for param := range in {
		// allow nil value to be written as series gaps
		if _, ok := param.Val.(float64); param.Val != nil && !ok {
			continue
		}

		p, err := influxdb.NewPoint(
			param.Key,
			map[string]string{
				"loadpoint": param.LoadPoint,
			},
			map[string]interface{}{
				"value": param.Val,
			},
			time.Now(),
		)
		if err != nil {
			m.log.ERROR.Printf("failed creating point: %v", err)
			continue
		}

		m.Lock()
		m.points = append(m.points, p)
		m.Unlock()
	}

	// close write loop
	exit <- struct{}{}
	<-done

	m.client.Close()
}
