package logx

import (
	"strings"
	"testing"

	kit "github.com/go-kit/log"
)

func TestRedaction(t *testing.T) {
	w := new(strings.Builder)

	base := kit.NewLogfmtLogger(kit.NewSyncWriter(w))
	log := Redact(base, "victorias", "secret")

	_ = log.Log("msg", "he knows victorias secrets and much more")
	exp := `msg="he knows *** ***s and much more"`

	if res := strings.TrimSpace(w.String()); !strings.EqualFold(res, exp) {
		t.Errorf("expected %s, got %s", exp, res)
	}
}
