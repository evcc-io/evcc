package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cast"
	"go.yaml.in/yaml/v4"
)

// parseLogLevels parses --log area:level[,...] switch into levels per log area
func parseLogLevels() {
	levels := viper.GetStringMapString("levels")
	if levels == nil {
		levels = make(map[string]string)
	}

	var level string
	for kv := range strings.SplitSeq(viper.GetString("log"), ",") {
		areaLevel := strings.SplitN(kv, ":", 2)
		if len(areaLevel) == 1 {
			level = areaLevel[0]
		} else {
			levels[areaLevel[0]] = areaLevel[1]
		}
	}

	util.LogLevel(level, levels)
}

// unwrap converts a wrapped error into slice of strings
func unwrap(err error) (res []string) {
	for err != nil {
		inner := errors.Unwrap(err)
		if inner == nil {
			res = append(res, err.Error())
		} else {
			cur := strings.TrimSuffix(err.Error(), ": "+inner.Error())
			cur = strings.TrimSuffix(cur, inner.Error())
			res = append(res, strings.TrimSpace(cur))
		}
		err = inner
	}
	return
}

// fatal logs a fatal error and runs shutdown functions before terminating
func fatal(err error) {
	log.FATAL.Println(err)
	<-shutdownDoneC()
	os.Exit(1)
}

// shutdownDoneC returns a channel that closes when shutdown has completed
func shutdownDoneC() <-chan struct{} {
	doneC := make(chan struct{})
	go shutdown.Cleanup(doneC)
	return doneC
}

// joinErrors is like errors.Join but does not wrap single errors (refs https://groups.google.com/g/golang-nuts/c/N0D1g5Ec_ZU)
func joinErrors(errs ...error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

func wrapFatalError(err error) error {
	if err == nil {
		return nil
	}

	if err2, ok := errors.AsType[*net.OpError](err); ok {
		if err2.Op == "listen" && strings.Contains(err2.Error(), "address already in use") {
			err = fmt.Errorf("could not open port- check that evcc is not already running (%w)", err)
		}
	}

	if err2, ok := errors.AsType[*os.PathError](err); ok {
		if err2.Op == "remove" && strings.Contains(err2.Error(), "operation not permitted") {
			err = fmt.Errorf("could not remove file- check that evcc is not already running (%w)", err)
		}
	}

	return &FatalError{err}
}

// customDevice promotes yaml type to top level type
func customDevice(typ string, other map[string]any) (string, map[string]any, error) {
	// no embedded yaml
	customYaml, ok := other["yaml"].(string)
	if !ok {
		return typ, other, nil
	}

	var res map[string]any
	if err := yaml.Unmarshal([]byte(customYaml), &res); err != nil {
		return typ, nil, err
	}

	// type override
	if typ := cast.ToString(res["type"]); typ != "" {
		delete(res, "type")
		return typ, res, nil
	}

	// no override
	return typ, res, nil
}

func deviceHeader[T any](dev config.Device[T]) string {
	name := dev.Config().Name

	if cd, ok := dev.(config.ConfigurableDevice[T]); ok {
		if title := cd.Properties().Title; title != "" {
			return fmt.Sprintf("%s (%s)", title, name)
		}
	}

	return name
}

// migrateYamlToJson converts a settings value from yaml to json if needed
func migrateYamlToJson[T any](key string, res *T) error {
	if settings.IsJson(key) {
		return settings.Json(key, res)
	}

	if err := settings.Yaml(key, new(T), res); err != nil {
		return err
	}

	return settings.SetJson(key, res)
}
