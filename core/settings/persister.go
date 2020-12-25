package settings

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andig/evcc/util"
	"github.com/spf13/viper"
)

type Storage struct {
	mux      sync.Mutex
	filename string
	cache    map[string]interface{}
	dirty    bool
}

func NewStorage(path string) *Storage {
	filename := fmt.Sprintf("%s/evcc", strings.TrimRight(path, "/"))

	return &Storage{
		filename: filename,
		cache:    make(map[string]interface{}),
	}
}

func (p *Storage) Load() {
	b, err := ioutil.ReadFile(p.filename)
	if err != nil {
		return
	}

	data := make(map[string]string)

	for _, val := range strings.Split(string(b), "\n") {
		val = strings.Trim(val, "\r\n")
		kv := strings.SplitN(val, ":", 2)

		data[kv[0]] = kv[1]
	}

	p.load(data)
}

func deepMerge(obj interface{}, path []string, val interface{}) interface{} {
	if len(path) == 0 {
		return val
	}

	el := path[0]

	if el, err := strconv.Atoi(el); err == nil {
		m, ok := obj.([]interface{})
		if !ok {
			if obj != nil {
				panic(fmt.Sprintf("expected slice type, got %T", obj))
			}
			m = make([]interface{}, 0)
		}

		for i := 0; i <= el; i++ {
			if len(m) <= i {
				m = append(m, nil)
			}
		}

		m[el] = deepMerge(m[el], path[1:], val)
		return m
	}

	m, ok := obj.(map[string]interface{})
	if !ok {
		if obj != nil {
			panic(fmt.Sprintf("expected map type, got %T", obj))
		}
		m = make(map[string]interface{})
	}
	m[el] = deepMerge(m[el], path[1:], val)

	return m
}

func (p *Storage) load(data map[string]string) {
	p.mux.Lock()
	defer p.mux.Unlock()

	for k, v := range data {
		path := strings.Split(k, ".")
		config := viper.Get(path[0])

		deepMerge(config, path[1:], v)
		viper.Set(path[0], config)
	}
}

func (p *Storage) Run(ch <-chan util.Param) {
	go func() {
		time.Sleep(5 * time.Second)
		p.Save()
	}()

	for param := range ch {
		key := param.UniqueID()

		if param.LoadPoint != nil {
			key = "loadpoints." + key
		}

		fmt.Printf("%s:%v\n", key, param.Val)

		p.mux.Lock()
		p.cache[key] = param.Val
		p.dirty = true
		p.mux.Unlock()
	}
}

func (p *Storage) copy() map[string]interface{} {
	m := make(map[string]interface{})

	for k, v := range p.cache {
		m[k] = v
	}

	return m
}

func (p *Storage) save(data map[string]interface{}) {
	buf := bytes.NewBuffer(nil)

	for k, v := range data {
		s := fmt.Sprintf("%s %v", k, v)
		if _, err := buf.WriteString(s); err != nil {
			return
		}
	}

	if err := ioutil.WriteFile(p.filename, buf.Bytes(), 0); err != nil {
		return
	}

	p.dirty = false
}

func (p *Storage) Save() {
	p.mux.Lock()
	defer p.mux.Unlock()

	if !p.dirty {
		return
	}

	data := p.copy()

	p.mux.Unlock()
	p.save(data)
	p.mux.Lock()
}
