// Command gridfees generates the grid-fees-de tariff template from the
// §14a EnWG grid fee data at https://github.com/ScumbagSteve/Grid-fees.
//
// Run from the repository root:
//
//	go run ./templates/gridfees
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	_ "embed"
)

const (
	sourceURL  = "https://codeload.github.com/ScumbagSteve/Grid-fees/tar.gz/refs/heads/main"
	outputPath = "templates/definition/tariff/grid-fees-de.yaml"
	unit       = "ct/kWh" // other units (e.g. percentage discounts) are not price zones
)

//go:embed grid-fees-de.tpl
var tmpl string

// source is the upstream per-operator file format
type source struct {
	GridOperator string `json:"grid_operator"`
	ValueUnit    string `json:"value_unit"`
	Years        map[string]struct {
		Fallback struct {
			Value *float64 `json:"value"`
		} `json:"fallback"`
		Periods []struct {
			DateFrom string `json:"date_from"`
			DateTo   string `json:"date_to"`
			Times    []struct {
				TimeFrom string   `json:"time_from"`
				TimeTo   string   `json:"time_to"`
				Value    *float64 `json:"value"`
			} `json:"times"`
		} `json:"periods"`
	} `json:"years"`
}

type zone struct {
	Price, Hours, Months string
}

type operator struct {
	Name, Price string
	Zones       []zone
}

func main() {
	files, err := download(sourceURL)
	if err != nil {
		log.Fatal(err)
	}

	var year int
	operators := make(map[string]operator)

	for _, name := range slices.Sorted(maps.Keys(files)) {
		var src source
		if err := json.Unmarshal(files[name], &src); err != nil {
			log.Fatalf("%s: %v", name, err)
		}

		op, y, err := convert(src)
		if err != nil {
			log.Printf("skipping %s: %v", src.GridOperator, err)
			continue
		}

		// upstream contains duplicate files per operator- keep the more detailed one
		if prev, ok := operators[op.Name]; !ok || windows(prev) < windows(op) {
			operators[op.Name] = op
		}
		year = max(year, y)
	}

	sorted := slices.SortedFunc(maps.Values(operators), func(i, j operator) int {
		return strings.Compare(i.Name, j.Name)
	})

	out, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	t := template.Must(template.New("tpl").Delims("[[", "]]").Parse(tmpl))
	if err := t.Execute(out, map[string]any{"Year": year, "Operators": sorted}); err != nil {
		log.Fatal(err)
	}

	log.Printf("wrote %s: %d grid operators, %d", outputPath, len(sorted), year)
}

// windows counts the operator's time windows
func windows(op operator) int {
	var res int
	for _, z := range op.Zones {
		res += strings.Count(z.Hours, ",") + 1
	}
	return res
}

// download returns the json files of the source repo archive
func download(uri string) (map[string][]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", uri, resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}

	res := make(map[string][]byte)

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return res, nil
		}
		if err != nil {
			return nil, err
		}

		if path.Ext(hdr.Name) != ".json" {
			continue
		}

		if res[path.Base(hdr.Name)], err = io.ReadAll(tr); err != nil {
			return nil, err
		}
	}
}

// convert converts the most recent year of a source file into template data
func convert(src source) (operator, int, error) {
	if src.ValueUnit != unit {
		return operator{}, 0, fmt.Errorf("unsupported unit: %s", src.ValueUnit)
	}

	years := make([]int, 0, len(src.Years))
	for y := range src.Years {
		i, err := strconv.Atoi(y)
		if err != nil {
			return operator{}, 0, err
		}
		years = append(years, i)
	}
	if len(years) == 0 {
		return operator{}, 0, fmt.Errorf("no years")
	}

	year := slices.Max(years)
	data := src.Years[strconv.Itoa(year)]

	if data.Fallback.Value == nil {
		return operator{}, 0, fmt.Errorf("%d: missing fallback", year)
	}
	if len(data.Periods) == 0 {
		return operator{}, 0, fmt.Errorf("%d: no periods", year)
	}

	// hours per price, months per identical set of price/hours
	hours := make(map[string]map[float64][]string)
	months := make(map[string][]string)
	var keys []string

	for _, p := range data.Periods {
		m, err := monthRange(p.DateFrom, p.DateTo)
		if err != nil {
			return operator{}, 0, fmt.Errorf("%d: %w", year, err)
		}

		byPrice := make(map[float64][]string)
		for _, t := range p.Times {
			if t.Value == nil {
				return operator{}, 0, fmt.Errorf("%d: missing value", year)
			}
			byPrice[*t.Value] = append(byPrice[*t.Value], t.TimeFrom+"-"+strings.Replace(t.TimeTo, "24:00", "00:00", 1))
		}

		key := fmt.Sprint(byPrice)
		if _, ok := hours[key]; !ok {
			hours[key] = byPrice
			keys = append(keys, key)
		}
		if m != "" {
			months[key] = append(months[key], m)
		}
	}

	op := operator{
		Name:  src.GridOperator,
		Price: price(*data.Fallback.Value),
	}

	for _, key := range keys {
		for _, p := range slices.Sorted(maps.Keys(hours[key])) {
			h := hours[key][p]
			slices.Sort(h)

			op.Zones = append(op.Zones, zone{
				Price:  price(p),
				Hours:  strings.Join(h, ","),
				Months: strings.Join(months[key], ","),
			})
		}
	}

	return op, year, nil
}

// price converts ct/kWh into currency/kWh
func price(v float64) string {
	return strings.TrimRight(strconv.FormatFloat(v/100, 'f', 6, 64), "0")
}

// monthRange converts a DD.MM. date range into a month range, empty for the entire year
func monthRange(from, to string) (string, error) {
	f, err := time.Parse("02.01.", from)
	if err != nil {
		return "", err
	}

	t, err := time.Parse("02.01.", to)
	if err != nil {
		return "", err
	}

	// zones are month- not day-based
	if f.Day() != 1 || t.Day() != time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day() {
		return "", fmt.Errorf("period is not month-aligned: %s-%s", from, to)
	}

	switch {
	case f.Month() == time.January && t.Month() == time.December:
		return "", nil
	case f.Month() == t.Month():
		return f.Format("Jan"), nil
	default:
		return f.Format("Jan") + "-" + t.Format("Jan"), nil
	}
}
