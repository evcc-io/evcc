package export

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/locale"
	"github.com/xuri/excelize/v2"
)

func TestWriteStructSliceXlsx_WithData(t *testing.T) {
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
	if err := WriteStructSliceXlsx(ctx, &buf, &slice, Config{I18nPrefix: "test.csv"}); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("output is not a valid xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected header + data row, got %d rows", len(rows))
	}

	if got := rows[0]; !equal(got, []string{"Name", "Value", "OptValue", "Created"}) {
		t.Errorf("unexpected header: %v", got)
	}
	if got := rows[1]; !equal(got[:3], []string{"Test", "123.456", "42.5"}) {
		t.Errorf("unexpected data row: %v", got)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
