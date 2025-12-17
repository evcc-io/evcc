package csv

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util/locale"
)

func init() {
	assets.I18n = os.DirFS("../../i18n")
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
	err := WriteStructSlice(ctx, &buf, &slice, Config{
		I18nPrefix: "test.csv",
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Name") {
		t.Errorf("Expected header with 'Name', got: %s", output)
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
	err := WriteStructSlice(ctx, &buf, &slice, Config{
		I18nPrefix: "test.csv",
	})

	if err != nil {
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

	expectedHeader := "Name,Value,OptValue,Created"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header: %s\nGot: %s", expectedHeader, lines[0])
	}

	if !strings.Contains(lines[1], "Test,123.456,42.5,2024-01-01") {
		t.Errorf("Expected data row to contain Test data\nGot: %s", lines[1])
	}
}

func TestWriteStructSlice_GermanLocale(t *testing.T) {
	var buf bytes.Buffer
	slice := TestStructs{
		{Name: "Test", Value: 123.456},
	}

	ctx := context.WithValue(context.Background(), locale.Locale, "de")
	err := WriteStructSlice(ctx, &buf, &slice, Config{
		I18nPrefix: "test.csv",
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")
	if len(lines) > 1 && !strings.Contains(lines[1], ";") {
		t.Errorf("Expected German CSV to use ';' separator, got: %s", lines[1])
	}
}

func TestFormatValue_NilPointer(t *testing.T) {
	var nilPtr *float64
	result := formatValue(nil, nilPtr, 3)
	if result != "" {
		t.Errorf("Expected empty string for nil pointer, got: %s", result)
	}
}

func TestFormatValue_ZeroTime(t *testing.T) {
	result := formatValue(nil, time.Time{}, 3)
	if result != "" {
		t.Errorf("Expected empty string for zero time, got: %s", result)
	}
}
