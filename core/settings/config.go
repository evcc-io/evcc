package settings

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

var _ Settings = (*ConfigSettings)(nil)

type ConfigSettings struct {
	accessor
	mu   sync.Mutex
	log  *util.Logger
	conf *config.Config
}

func NewConfigSettingsAdapter(log *util.Logger, conf *config.Config) *ConfigSettings {
	s := &ConfigSettings{log: log, conf: conf}
	s.accessor = accessor{s.get, s.set}
	return s
}

func (s *ConfigSettings) get(key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val := s.conf.Named().Other[key]
	if val == nil {
		return nil, errors.New("not found")
	}
	return val, nil
}

// TODO remove broken error handling when settings api is retired
func (s *ConfigSettings) set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.conf.Named().Other
	data[key] = val
	if err := s.conf.Update(data); err != nil {
		s.log.ERROR.Println(err)
	}
}

func (s *ConfigSettings) SetJson(key string, val any) error {
	s.set(key, val)
	return nil
}

func (s *ConfigSettings) Json(key string, res any) error {
	str, err := s.String(key)
	if str == "" || err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), &res)
}
