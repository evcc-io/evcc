package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	uri, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	doc, err := openapi3.NewLoader().LoadFromURI(uri)
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
