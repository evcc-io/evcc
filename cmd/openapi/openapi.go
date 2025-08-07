package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	doc, err := openapi3.NewLoader().LoadFromFile(os.Args[1])
	if err != nil {
		log.Fatal("failed to load OpenAPI spec:", err)
	}

	// omit servers
	doc.Servers = nil

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(os.Args[2], b, 0o644); err != nil {
		log.Fatal(err)
	}
}
