package server

import (
	"crypto/tls"
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
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	influxlog "github.com/influxdata/influxdb-client-go/v2/log"
)

// Influx is a influx publisher
type Influx struct {
	sync.Mutex
	log      *util.Logger
	clock    clock.Clock
	client   influxdb2.Client
	org      string
	database string
}

// NewInfluxClient creates new publisher for influx
func NewInfluxClient(url, token, org, user, password, database string, insecure bool) *Influx {
	log := util.NewLogger("influx")

	// InfluxDB v1 compatibility
	if token == "" && user != "" {
		token = fmt.Sprintf("%s:%s", user, password)
	}

	options := influxdb2.DefaultOptions()

	options.SetTLSConfig(&tls.Config{InsecureSkipVerify: insecure})
	options.SetPrecision(time.Second)

	client := influxdb2.NewClientWithOptions(url, token, options)

	// handle error logging in writer
	influxlog.Log = nil

	return &Influx{
		log:      log,
		clock:    clock.New(),
		client:   client,
		org:      org,
		database: database,
	}
}

// pointWriter is the minimal interface for influxdb2 api.Writer
type pointWriter interface {
	WritePoint(point *write.Point)
}

// writePoint asynchronously writes a point to influx
func (m *Influx) writePoint(writer pointWriter, key string, fields map[string]any, tags map[string]string) {
	m.log.TRACE.Printf("write %s=%v (%v)", key, fields, tags)
	writer.WritePoint(influxdb2.NewPoint(key, tags, fields, m.clock.Now()))
}

// writeComplexPoint asynchronously writes a point to influx
func (m *Influx) writeComplexPoint(writer pointWriter, key string, val any, tags map[string]string) {
	fields := make(map[string]any)

	// loop struct
	writeStruct := func(sv any) {
		typ := reflect.TypeOf(sv)
		val := reflect.ValueOf(sv)

		for i := range typ.NumField() {
			if f := typ.Field(i); f.IsExported() {
				if val.Field(i).IsZero() && omitEmpty(f) {
					continue
				}

				key := key + strings.ToUpper(f.Name[:1]) + f.Name[1:]
				val := val.Field(i).Interface()

				m.writeComplexPoint(writer, key, val, tags)
			}
		}
	}

	switch valueType := val.(type) {
	case string:
		return

	case int, int64, float64:
		fields["value"] = val

	case []float64:
		if len(valueType) != 3 {
			return
		}

		// add array as phase values
		for i, v := range valueType {
			fields[fmt.Sprintf("l%d", i+1)] = v
		}

	case [3]float64:
		// add array as phase values
		for i, v := range valueType {
			fields[fmt.Sprintf("l%d", i+1)] = v
		}

	default:
		// allow writing nil values
		if val == nil {
			fields["value"] = nil
			break
		}

		switch typ := reflect.TypeOf(val); {
		// pointer
		case typ.Kind() == reflect.Ptr:
			if val := reflect.ValueOf(val); !val.IsNil() {
				m.writeComplexPoint(writer, key, reflect.Indirect(val).Interface(), tags)
			}

		// struct
		case typ.Kind() == reflect.Struct:
			writeStruct(val)

		// slice of structs
		case typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Struct:
			val := reflect.ValueOf(val)

			// loop slice
			for i := range val.Len() {
				tags["id"] = strconv.Itoa(i + 1)
				writeStruct(val.Index(i).Interface())
			}
		}

		return
	}

	m.writePoint(writer, key, fields, tags)
}

// Run Influx publisher
func (m *Influx) Run(site site.API, in <-chan util.Param) {
	writer := m.client.WriteAPI(m.org, m.database)

	// log errors
	go func() {
		for err := range writer.Errors() {
			// log async as we're part of the logging loop
			go m.log.ERROR.Println(err)
		}
	}()

	// add points to batch for async writing
	for param := range in {
		tags := make(map[string]string)
		if param.Loadpoint != nil {
			lp := site.Loadpoints()[*param.Loadpoint]

			tags["loadpoint"] = lp.GetTitle()
			if v := lp.GetVehicle(); v != nil {
				tags["vehicle"] = v.Title()
			}
		}

		m.writeComplexPoint(writer, param.Key, param.Val, tags)
	}

	m.client.Close()
}
