package porsche

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeSkipsPasskeyEnrollment(t *testing.T) {
	ctx := base64.StdEncoding.EncodeToString([]byte(`{"transaction":{"state":"tx"},"untrustedData":{"submittedFormData":{"foo":"bar"}}}`))

	mux := http.NewServeMux()
	mux.HandleFunc("/authorize/resume", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, passkeyEnrollmentPath+"?state=tx", http.StatusFound)
	})
	mux.HandleFunc(passkeyEnrollmentPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`<script>atob("` + ctx + `")</script>`))
			return
		}
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "tx", r.PostForm.Get("state"))
		assert.Equal(t, "abort-passkey-enrollment", r.PostForm.Get("action"))
		assert.Equal(t, "bar", r.PostForm.Get("foo"))
		w.Header().Set("Location", RedirectURI+"?code=abc&state=evcc")
		w.WriteHeader(http.StatusFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := &LoginSession{client: &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
	code, err := s.resume(srv.URL + "/authorize/resume")
	require.NoError(t, err)
	require.Equal(t, "abc", code)
}
