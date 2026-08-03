package util

import (
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/evcc-io/evcc/util/encode"
)

// Param is the broadcast channel data type
type Param struct {
	Loadpoint *int
	Vehicle   *string
	Key       string
	Val       any
}

// UniqueID returns unique identifier for parameter Loadpoint/Key combination
func (p Param) UniqueID() string {
	if p.Loadpoint != nil {
		return strconv.Itoa(*p.Loadpoint) + "." + p.Key
	} else if p.Vehicle != nil {
		return *p.Vehicle + "." + p.Key
	}

	return p.Key
}

// ParamCache is a data store
type ParamCache struct {
	mu  sync.RWMutex
	val map[string]Param
}

// Snapshot requests a copy of the cache state at the parameter's position in
// the stream. It runs on the cache's goroutine and must not block for long.
type Snapshot func([]Param)

// NewCache creates cache
func NewParamCache() *ParamCache {
	return &ParamCache{
		val: make(map[string]Param),
	}
}

// Run adds input channel's values to cache
func (c *ParamCache) Run(in <-chan Param) {
	for p := range in {
		if snapshot, ok := p.Val.(Snapshot); ok {
			snapshot(c.All())
			continue
		}

		c.Add(p.UniqueID(), p)
	}
}

// State provides a structured copy of the cached values.
// Loadpoints are aggregated as loadpoints array.
// Result values are formatted using encoder.
func (c *ParamCache) State(enc encode.Encoder) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make(map[string]any)
	lps := make(map[int]map[string]any)
	vDyn := make(map[string]map[string]any)

	for _, param := range c.val {
		if param.Loadpoint != nil {
			lp, ok := lps[*param.Loadpoint]
			if !ok {
				lp = make(map[string]any)
				lps[*param.Loadpoint] = lp
			}
			lp[param.Key] = enc.Encode(param.Val)
		} else if param.Vehicle != nil {
			v, ok := vDyn[*param.Vehicle]
			if !ok {
				v = make(map[string]any)
				vDyn[*param.Vehicle] = v
			}
			v[param.Key] = enc.Encode(param.Val)
		} else {
			res[param.Key] = enc.Encode(param.Val)
		}
	}

	// convert map to array
	loadpoints := make([]map[string]any, len(lps))
	for id, lp := range lps {
		loadpoints[id] = lp
	}
	res["loadpoints"] = loadpoints

	MergeVehicleState(res, vDyn)

	return res
}

// All provides a copy of the cached values
func (c *ParamCache) All() []Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return slices.Collect(maps.Values(c.val))
}

// Add entry to cache
func (c *ParamCache) Add(key string, param Param) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.val[key] = param
}

// Get entry from cache
func (c *ParamCache) Get(key string) Param {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.val[key]; ok {
		return val
	}

	return Param{}
}

func MergeVehicleState(res map[string]any, vehiclesDyn map[string]map[string]any) {
	vm := convertVehicleStructMap(res["vehicles"])
	res["vehicles"] = vm

	for vid, dyn := range vehiclesDyn {
		vcfg, ok := vm[vid].(map[string]any)
		if !ok {
			vcfg = make(map[string]any)
			vm[vid] = vcfg
		}

		for k, v := range dyn {
			vcfg[k] = v
		}
	}
}

func convertVehicleStructMap(in any) map[string]any {
	out := make(map[string]any)

	rv := reflect.ValueOf(in)
	for _, key := range rv.MapKeys() {
		v := rv.MapIndex(key)

		// struct → map[string]any
		m := make(map[string]any)
		st := v.Interface()

		sv := reflect.ValueOf(st)
		stt := sv.Type()

		for i := 0; i < stt.NumField(); i++ {
			field := stt.Field(i)
			name := field.Name
			value := sv.Field(i).Interface()

			// nur exportierte Felder
			if field.PkgPath == "" {
				m[strings.ToLower(name[:1])+name[1:]] = value
			}
		}

		out[key.String()] = m
	}

	return out
}
