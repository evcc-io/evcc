package settings

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/andig/evcc/util"
)

type Storage struct {
	mux      sync.Mutex
	filename string
	cache    map[string]interface{}
	dirty    bool
}

func NewStorage(filename string) *Storage {
	return &Storage{
		filename: filename,
		cache:    make(map[string]interface{}),
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
