package server

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	influxlog "github.com/influxdata/influxdb-client-go/v2/log"
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
	log    *util.Logger
	clock  clock.Clock
	client influxdb2.Client
	writer influxapi.WriteAPI
}

// NewInfluxClient creates new publisher for influx
func NewInfluxClient(url, token, org, user, password, database string) *Influx {
	log := util.NewLogger("influx")

	// InfluxDB v1 compatibility
	if token == "" && user != "" {
		token = fmt.Sprintf("%s:%s", user, password)
	}

	options := influxdb2.DefaultOptions().SetPrecision(time.Second)
	client := influxdb2.NewClientWithOptions(url, token, options)

	// handle error logging in writer
	influxlog.Log = nil

	writer := client.WriteAPI(org, database)

	// log errors
	go func() {
		for err := range writer.Errors() {
			// log async as we're part of the logging loop
			go log.ERROR.Println(err)
		}
	}()

	return &Influx{
		log:    log,
		clock:  clock.New(),
		client: client,
		writer: writer,
	}
}

// writePoint asynchronously writes a point to influx
func (m *Influx) writePoint(key string, fields map[string]any, tags map[string]string) {
	m.log.TRACE.Printf("write %s=%v (%v)", key, fields, tags)
	m.writer.WritePoint(influxdb2.NewPoint(key, tags, fields, m.clock.Now()))
}

// writeComplexPoint asynchronously writes a point to influx
func (m *Influx) writeComplexPoint(param util.Param, tags map[string]string) {
	fields := make(map[string]any)

	switch val := param.Val.(type) {
	case string:
		return

	case int, int64, float64:
		fields["value"] = param.Val

	case []float64:
		if len(val) != 3 {
			return
		}

		// add array as phase values
		for i, v := range val {
			fields[fmt.Sprintf("l%d", i+1)] = v
		}

	case [3]float64:
		// add array as phase values
		for i, v := range val {
			fields[fmt.Sprintf("l%d", i+1)] = v
		}

	default:
		// allow writing nil values
		if param.Val == nil {
			fields["value"] = nil
			break
		}

		// slice of structs
		if typ := reflect.TypeOf(param.Val); typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Struct {
			val := reflect.ValueOf(param.Val)

			// loop slice
			for i := 0; i < val.Len(); i++ {
				val := val.Index(i)
				typ := val.Type()

				// loop struct
				for j := 0; j < typ.NumField(); j++ {
					n := typ.Field(j).Name
					v := val.Field(j).Interface()

					key := param.Key + strings.ToUpper(n[:1]) + n[1:]
					fields["value"] = v
					tags["id"] = strconv.Itoa(i + 1)

					m.writePoint(key, fields, tags)
				}
			}
		}

		return
	}

	m.writePoint(param.Key, fields, tags)
}

// Run Influx publisher
func (m *Influx) Run(site site.API, in <-chan util.Param) {
	// add points to batch for async writing
	for param := range in {
		tags := make(map[string]string)
		if param.Loadpoint != nil {
			lp := site.Loadpoints()[*param.Loadpoint]

			tags["loadpoint"] = lp.Title()
			if v := lp.GetVehicle(); v != nil {
				tags["vehicle"] = v.Title()
			}
		}

		m.writeComplexPoint(param, tags)
	}

	m.client.Close()
}
