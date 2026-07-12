package csv

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util/export"
	"github.com/evcc-io/evcc/util/locale"
)

func init() {
	assets.I18n = os.DirFS("../../../i18n")
	_ = locale.Init()
}

type TestStruct struct {
	ID       int       `json:"id" csv:"-"`
	Name     string    `json:"name"`
	Value    float64   `json:"value"`
	OptValue *float64  `json:"optValue,omitempty"`
	Created  time.Time `json:"created"`
}

type TestStructs []TestStruct

func TestWriteStructSlice_Empty(t *testing.T) {
	var buf bytes.Buffer
	slice := TestStructs{}

	ctx := context.WithValue(context.Background(), locale.Locale, "en")
	if err := WriteStructSlice(ctx, &buf, &slice, export.Config{I18nPrefix: "test.csv"}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !strings.Contains(buf.String(), "Name") {
		t.Errorf("Expected header with 'Name', got: %s", buf.String())
	}
}

func TestWriteStructSlice_WithData(t *testing.T) {
	var buf bytes.Buffer
	val := 42.5
	slice := TestStructs{
		{
			ID:       1,
			Name:     "Test",
			Value:    123.456,
			OptValue: &val,
			Created:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	ctx := context.WithValue(context.Background(), locale.Locale, "en")
	if err := WriteStructSlice(ctx, &buf, &slice, export.Config{I18nPrefix: "test.csv"}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "\xEF\xBB\xBF") {
		t.Error("Expected UTF-8 BOM at start")
	}

	lines := strings.Split(strings.TrimPrefix(output, "\xEF\xBB\xBF"), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines (header + data), got: %d", len(lines))
	}

	if lines[0] != "Name,Value,OptValue,Created" {
		t.Errorf("Expected header: Name,Value,OptValue,Created\nGot: %s", lines[0])
	}

	if !strings.Contains(lines[1], "Test,123.456,42.5,2024-01-01") {
		t.Errorf("Expected data row to contain Test data\nGot: %s", lines[1])
	}
}

func TestWriteStructSlice_LocaleIndependentValues(t *testing.T) {
	var buf bytes.Buffer
	slice := TestStructs{
		{Name: "Test", Value: 123.456},
	}

	ctx := context.WithValue(context.Background(), locale.Locale, "de")
	if err := WriteStructSlice(ctx, &buf, &slice, export.Config{I18nPrefix: "test.csv"}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	lines := strings.Split(buf.String(), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected header + data row, got: %s", buf.String())
	}
	if !strings.Contains(lines[1], "Test,123.456") {
		t.Errorf("Expected comma delimiter and dot decimal for German locale, got: %s", lines[1])
	}
}
