package export

type Config struct {
	I18nPrefix string // e.g., "sessions.csv" or "config.hems.csv"
}

// RowWriter is the row-oriented sink shared by the CSV and XLSX exporters.
// *csv.Writer satisfies it directly.
type RowWriter interface {
	Write(record []string) error
	Flush()
	Error() error
}
