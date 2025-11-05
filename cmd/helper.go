package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
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

func redact(src string) string {
	return util.RedactConfigString(src)
}

func redactMap(src map[string]any) map[string]any {
	return util.RedactConfigMap(src)
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

	var opErr *net.OpError
	var pathErr *os.PathError

	switch {
	case errors.As(err, &opErr):
		if opErr.Op == "listen" && strings.Contains(opErr.Error(), "address already in use") {
			err = fmt.Errorf("could not open port- check that evcc is not already running (%w)", err)
		}

	case errors.As(err, &pathErr):
		if pathErr.Op == "remove" && strings.Contains(pathErr.Error(), "operation not permitted") {
			err = fmt.Errorf("could not remove file- check that evcc is not already running (%w)", err)
		}
	}

	return &FatalError{err}
}

func customDevice(other map[string]any) (map[string]any, error) {
	customYaml, ok := other["yaml"].(string)
	if !ok {
		return other, nil
	}

	var res map[string]any
	err := yaml.Unmarshal([]byte(customYaml), &res)
	return res, err
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
