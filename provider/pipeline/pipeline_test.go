package pipeline

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegex(t *testing.T) {
	for _, re := range []string{`([0-9.]+)`, `[0-9.]+`} {
		p, err := new(Pipeline).WithRegex(re, "")
		require.NoError(t, err)

		res, err := p.Process([]byte("12.3W"))
		require.NoError(t, err)

		require.Equal(t, []byte("12.3"), res)
	}
}

func TestRegexDefault(t *testing.T) {
	p, err := new(Pipeline).WithRegex(`\d+`, "123")
	require.NoError(t, err)

	res, err := p.Process([]byte("xxx"))
	require.NoError(t, err)

	require.Equal(t, []byte("123"), res)
}

func TestJq(t *testing.T) {
	for _, uuid := range []string{"a8232ee0-a4ab-11ec-8d36-211f6b082dc8", "08232ee0-a4ab-11ec-8d36-211f6b082dc8"} {

		p, err := new(Pipeline).WithJq(fmt.Sprintf(`.data[] | select(.uuid=="%s") | .tuples[0][1]`, uuid))
		require.NoError(t, err)

		res, err := p.Process([]byte(`
			{
				"data": [
					{
						"uuid": "` + uuid + `",
						"tuples": [[1,2,3]]
					}
				]
			}
		`))
		require.NoError(t, err)

		require.Equal(t, []byte("2"), res)
	}
}
