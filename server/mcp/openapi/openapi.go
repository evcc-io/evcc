package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/evcc-io/evcc/server"
	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	doc, err := openapi3.NewLoader().LoadFromData(server.OpenAPI)
	if err != nil {
		log.Fatal("failed to load OpenAPI spec:", err)
	}

	// omit servers
	doc.Servers = nil

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("openapi.json", b, 0o644); err != nil {
		log.Fatal(err)
	}

	log.Println("OpenAPI spec written to openapi.json")
}
