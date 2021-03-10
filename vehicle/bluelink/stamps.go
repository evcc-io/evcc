package bluelink

import (
	"bufio"
	_ "embed" // embedded files
	"math/rand"
	"strings"
)

//go:embed 693a33fa-c117-43f2-ae3b-61a02d24f417
var kia string

//go:embed 99cfff84-f4e2-4be8-a5ed-e5b755eb6581
var hyundai string

// Stamps collects stamps for a single brand
type Stamps map[string][]string

var stamps = Stamps{
	"693a33fa-c117-43f2-ae3b-61a02d24f417": unpack(kia),
	"99cfff84-f4e2-4be8-a5ed-e5b755eb6581": unpack(hyundai),
}

func unpack(source string) (res []string) {
	scanner := bufio.NewScanner(strings.NewReader(source))
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return
}

// New creates a new stamp
func (s Stamps) New(id string) string {
	source, ok := s[id]
	if !ok {
		panic(id)
	}
	return source[rand.Intn(len(source))]
}
