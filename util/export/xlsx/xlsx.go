package xlsx

import (
	"context"
	"io"
	"time"

	"github.com/evcc-io/evcc/util/export"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/xuri/excelize/v2"
)

// writer adapts excelize to export.RowWriter with native typed cells
type writer struct {
	f         *excelize.File
	sheet     string
	dst       io.Writer
	row       int
	err       error
	localizer *i18n.Localizer
	dateStyle int
	durStyle  int
}

// cellStyle returns the number format style for typed cells, 0 for default
func (x *writer) cellStyle(v any) int {
	switch v.(type) {
	case time.Time:
		return x.dateStyle
	case time.Duration:
		return x.durStyle
	default:
		return 0
	}
}

func (x *writer) Write(record []any) error {
	for i, v := range record {
		if v == nil {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(i+1, x.row)
		if err != nil {
			x.err = err
			return err
		}
		if err := x.f.SetCellValue(x.sheet, cell, v); err != nil {
			x.err = err
			return err
		}
		if style := x.cellStyle(v); style != 0 {
			if err := x.f.SetCellStyle(x.sheet, cell, cell, style); err != nil {
				x.err = err
				return err
			}
		}
	}
	if x.row == 1 {
		if err := x.styleHeader(len(record)); err != nil {
			x.err = err
			return err
		}
	}
	x.row++
	return nil
}

// styleHeader makes the header row bold and freezes it below the viewport.
func (x *writer) styleHeader(cols int) error {
	if cols == 0 {
		return nil
	}
	style, err := x.f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	last, err := excelize.CoordinatesToCellName(cols, 1)
	if err != nil {
		return err
	}
	if err := x.f.SetCellStyle(x.sheet, "A1", last, style); err != nil {
		return err
	}
	return x.f.SetPanes(x.sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
}

func (x *writer) Flush() {
	if x.err != nil {
		return
	}
	if err := x.f.Write(x.dst); err != nil {
		x.err = err
	}
}

func (x *writer) Error() error               { return x.err }
func (x *writer) Localizer() *i18n.Localizer { return x.localizer }

// New returns an xlsx-backed export.RowWriter. The context locale only affects header captions.
func New(ctx context.Context, w io.Writer) (export.RowWriter, error) {
	f := excelize.NewFile()

	dateFmt, durFmt := "yyyy-mm-dd hh:mm:ss", "[h]:mm:ss"
	dateStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &dateFmt})
	if err != nil {
		return nil, err
	}
	durStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &durFmt})
	if err != nil {
		return nil, err
	}

	return &writer{
		f:         f,
		sheet:     f.GetSheetName(0),
		dst:       w,
		row:       1,
		localizer: export.Localizer(ctx),
		dateStyle: dateStyle,
		durStyle:  durStyle,
	}, nil
}

// WriteStructSlice writes a slice of structs to xlsx with localized headers.
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg export.Config) error {
	ww, err := New(ctx, w)
	if err != nil {
		return err
	}
	return export.WriteStructSlice(ww, slice, cfg)
}
