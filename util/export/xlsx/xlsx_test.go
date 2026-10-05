package xlsx

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
	"github.com/xuri/excelize/v2"
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

	sheet := f.GetSheetName(0)

	// numeric cells carry no type attribute, strings are shared strings
	if typ, err := f.GetCellType(sheet, "B2"); err != nil || typ == excelize.CellTypeSharedString {
		t.Errorf("expected numeric cell for Value, got type %v, err %v", typ, err)
	}
	if v, err := f.GetCellValue(sheet, "D2", excelize.Options{RawCellValue: true}); err != nil || v == "" || strings.Contains(v, "-") {
		t.Errorf("expected date serial number for Created, got %q, err %v", v, err)
	}

	styleID, err := f.GetCellStyle(sheet, "A1")
	if err != nil {
		t.Fatalf("GetCellStyle: %v", err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle: %v", err)
	}
	if style.Font == nil || !style.Font.Bold {
		t.Error("expected bold header")
	}

	panes, err := f.GetPanes(sheet)
	if err != nil {
		t.Fatalf("GetPanes: %v", err)
	}
	if !panes.Freeze {
		t.Error("expected frozen header row")
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
