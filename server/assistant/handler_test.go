package assistant

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// an unconfigured assistant answers every question the same way, whatever the
// body says. The check precedes parsing, so no request reaches the model
func TestChatHandlerUnconfigured(t *testing.T) {
	for _, body := range []string{
		`{"messages":[{"role":"user","content":"hi"}]}`,
		`{"messages":[]}`,
		`{`,
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(body))

		ChatHandler(http.NotFoundHandler())(w, r)

		assert.Equal(t, http.StatusPreconditionFailed, w.Code, body)
		assert.Contains(t, w.Body.String(), "error", body)
	}
}
