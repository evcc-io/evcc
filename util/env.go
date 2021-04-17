package util

import (
	"log"
	"os"
	"strings"
)

func Getenv(key string, def ...string) string {
	res := strings.TrimSpace(os.Getenv(key))
	if res == "" {
		if len(def) == 1 {
			return def[0]
		}

		log.Fatalln("missing", key)
	}
	return res
}
