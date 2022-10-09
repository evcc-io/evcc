package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setLogLevel sets log level from config overwritten by command line
// https://github.com/spf13/viper/issues/1444
func setLogLevel(cmd *cobra.Command) {
	if flag := cmd.Flags().Lookup("log"); viper.GetString("log") == "" || flag.Changed {
		viper.Set("log", flag.Value.String())
	}
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
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

// redact redacts a configuration string
func redact(src string) string {
	secrets := []string{
		"url", "uri", "host", "broker", "mac", // infrastructure
		"sponsortoken", "plant", // global settings
		"user", "password", "pin", // users
		"token", "access", "refresh", // tokens
		"ain", "id", "secret", "serial", "deviceid", "machineid", // devices
		"vin"} // vehicles
	return regexp.
		MustCompile(fmt.Sprintf(`\b(%s)\b.*?:.*`, strings.Join(secrets, "|"))).
		ReplaceAllString(src, "$1: *****")
}

func publishErrorInfo(cfgFile string, err error) {
	if cfgFile != "" {
		file, pathErr := filepath.Abs(cfgFile)
		if pathErr != nil {
			file = cfgFile
		}
		publish("file", file)

		if src, fileErr := os.ReadFile(cfgFile); fileErr != nil {
			log.ERROR.Println("could not open config file:", fileErr)
		} else {
			publish("config", redact(string(src)))

			// find line number
			if match := regexp.MustCompile(`yaml: line (\d+):`).FindStringSubmatch(err.Error()); len(match) == 2 {
				if line, err := strconv.Atoi(match[1]); err == nil {
					publish("line", line)
				}
			}
		}
	}

	publish("fatal", unwrap(err))
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

// exitWhenDone waits for shutdown to complete with timeout
func exitWhenDone(timeout time.Duration) {
	select {
	case <-shutdownDoneC(): // wait for shutdown
	case <-time.After(timeout):
	}

	os.Exit(1)
}

// exitWhenStopped waits for stop and performs shutdown
func exitWhenStopped(stopC <-chan struct{}, timeout time.Duration) {
	<-stopC
	exitWhenDone(timeout)
}
