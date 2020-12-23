package main

// moved to Makefile for splitting backend and frontend build
// go:generate esc -o server/assets.go -pkg server -modtime 1566640112 -ignore .DS_Store dist

import (
	"github.com/andig/evcc/cmd"
)

func main() {
	cmd.Execute()
}
