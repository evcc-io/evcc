package metrics

import (
	"database/sql"
	"errors"
	"time"
)

type SqlTime time.Time

var _ sql.Scanner = (*SqlTime)(nil)

func (st *SqlTime) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		*st = SqlTime(v)
	case int64:
		*st = SqlTime(time.Unix(v, 0))
	case string:
		t, err := time.Parse(time.DateTime+"-07:00", v)
		if err == nil {
			*st = SqlTime(t)
		}
		return err
	default:
		return errors.New("unsupported timestamp type")
	}
	return nil
}
