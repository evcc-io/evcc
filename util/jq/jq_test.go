package jq

import (
	"context"
	"testing"
	"time"

	"github.com/itchyny/gojq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	query, err := gojq.Parse(".foo")
	require.NoError(t, err)

	res, err := Query(query, []byte(`{"foo": 42}`))
	require.NoError(t, err)
	assert.Equal(t, float64(42), res)
}

// TestQueryContextTimeout ensures non-terminating queries are aborted instead of
// running forever, exhausting cpu or memory
func TestQueryContextTimeout(t *testing.T) {
	for _, q := range []string{
		`[range(1e9)]`, // unbounded allocation
		`def f: f; f`,  // unbounded recursion
		`[repeat(0)]`,  // infinite generator
	} {
		t.Run(q, func(t *testing.T) {
			query, err := gojq.Parse(q)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, err := QueryContext(ctx, query, []byte(`{}`))
				done <- err
			}()

			select {
			case err := <-done:
				assert.Error(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("query did not terminate")
			}
		})
	}
}

func TestQueryContextHalt(t *testing.T) {
	query, err := gojq.Parse(`halt`)
	require.NoError(t, err)

	_, err = Query(query, []byte(`{}`))
	assert.Error(t, err)
}
