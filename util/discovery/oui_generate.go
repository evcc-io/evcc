//go:build ignore

// Converts IEEE registry csv files into the embedded vendor list
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

func main() {
	var lines []string

	for _, file := range os.Args[1:] {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}

		records, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			log.Fatal(err)
		}

		// Registry,Assignment,Organization Name,Organization Address
		for _, r := range records[1:] {
			if name := strings.Join(strings.Fields(r[2]), " "); name != "" && name != "Private" && name != "IEEE Registration Authority" {
				lines = append(lines, fmt.Sprintf("%s\t%s", strings.ToUpper(r[1]), name))
			}
		}
	}

	slices.Sort(lines)
	lines = slices.Compact(lines)

	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := w.Write([]byte(strings.Join(lines, "\n"))); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("oui.txt.gz", buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
