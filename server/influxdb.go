package server

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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
	log      *util.Logger
	client   influxdb2.Client
	org      string
	database string
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

	return &Influx{
		log:      log,
		client:   client,
		org:      org,
		database: database,
	}
}

// Run Influx publisher
func (m *Influx) Run(loadPoints []loadpoint.API, in <-chan util.Param) {
	writer := m.client.WriteAPI(m.org, m.database)

	// log errors
	go func() {
		for err := range writer.Errors() {
			// log async as we're part of the logging loop
			go m.log.ERROR.Println(err)
		}
	}()

	// track active vehicle per loadpoint
	vehicles := make(map[int]string)

	// add points to batch for async writing
	for param := range in {
		// vehicle name
		if param.Loadpoint != nil && param.Key == "vehicleTitle" {
			if vehicle, ok := param.Val.(string); ok {
				vehicles[*param.Loadpoint] = vehicle
				continue
			}
		}

		fields := make(map[string]interface{})

		tags := make(map[string]string)
		if param.Loadpoint != nil {
			tags["loadpoint"] = loadPoints[*param.Loadpoint].Name()
			tags["vehicle"] = vehicles[*param.Loadpoint]
		}

		switch val := param.Val.(type) {
		case int, int64, float64:
			fields["value"] = param.Val

		case [3]float64:
			// add array as phase values
			for i, v := range val {
				fields[fmt.Sprintf("l%d", i+1)] = v
			}

		default:
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

						key := param.Key + strings.ToLower(n[:1]) + n[1:]
						fields["value"] = v
						tags["id"] = strconv.Itoa(i + 1)

						// write asynchronously
						m.log.TRACE.Printf("write %s=%v (%v)", key, fields, tags)
						p := influxdb2.NewPoint(key, tags, fields, time.Now())
						writer.WritePoint(p)
					}
				}

				continue
			}

			// any other unsupported type
			if param.Val != nil {
				continue
			}
		}

		// write asynchronously
		m.log.TRACE.Printf("write %s=%v (%v)", param.Key, fields, tags)
		p := influxdb2.NewPoint(param.Key, tags, fields, time.Now())
		writer.WritePoint(p)
	}

	m.client.Close()
}
