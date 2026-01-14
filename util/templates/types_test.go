package templates

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParamLogic(t *testing.T) {
	{
		p := Param{
			Advanced: true, // omitempty
			Required: true,
		}
		require.False(t, p.IsAdvanced(), "can't be advanced when required")

		{
			b, err := json.Marshal(p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"","Help":"","Required":true,"Type":"String"}`, string(b), "Marshal p advanced/required")
		}
		{
			b, err := json.Marshal(&p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"","Help":"","Required":true,"Type":"String"}`, string(b), "Marshal *p advanced/required")
		}
	}

	{
		p := Param{
			Deprecated: true,
			Required:   true, // omitempty
		}
		require.False(t, p.IsRequired(), "can't be required when deprecated")

		{
			b, err := json.Marshal(p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"","Help":"","Deprecated":true,"Type":"String"}`, string(b), "Marshal p deprecated/required")
		}
		{
			b, err := json.Marshal(&p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"","Help":"","Deprecated":true,"Type":"String"}`, string(b), "Marshal *p deprecated/required")
		}
	}

	b, err := json.Marshal(Param{
		Deprecated: true,
		Advanced:   true, // omitempty
		Required:   true, // omitempty
	})
	require.NoError(t, err)
	require.Equal(t, `{"Name":"","Description":"","Help":"","Deprecated":true,"Type":"String"}`, string(b))
}

func TestParamMarshal(t *testing.T) {
	{
		p := Param{
			Description: TextLanguage{Generic: "foo"},
		}

		{
			b, err := json.Marshal(p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"foo","Help":"","Type":"String"}`, string(b))
		}

		{
			b, err := json.Marshal(&p)
			require.NoError(t, err)
			assert.Equal(t, `{"Name":"","Description":"foo","Help":"","Type":"String"}`, string(b))
		}
	}
}
