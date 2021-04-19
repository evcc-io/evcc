package main

import (
	_ "embed"
	"encoding/base64"
	"fmt"
)

//go:embed server/server-key.pem
var key []byte

func main() {
	fmt.Println(base64.StdEncoding.EncodeToString(key))
}
