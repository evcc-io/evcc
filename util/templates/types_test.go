package templates

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamLogic(t *testing.T) {
	{
		p := Param{
			Advanced: true, // omitempty
			Required: true,
		}
		require.False(t, p.IsAdvanced(), "can't be advanced when required")

		b, err := json.Marshal(p)
		require.NoError(t, err)
		require.Equal(t, `{"Name":"","Description":{},"Help":{},"Required":true,"Type":"String"}`, string(b))
	}

	{
		p := Param{
			Deprecated: true,
			Required:   true,
		}
		require.False(t, p.IsRequired(), "can't be required when deprecated")

		b, err := json.Marshal(Param{
			Deprecated: true,
			Required:   true, // omitempty
		})
		require.NoError(t, err)
		require.Equal(t, `{"Name":"","Description":{},"Help":{},"Deprecated":true,"Type":"String"}`, string(b))
	}

	b, err := json.Marshal(Param{
		Deprecated: true,
		Advanced:   true, // omitempty
		Required:   true, // omitempty
	})
	require.NoError(t, err)
	require.Equal(t, `{"Name":"","Description":{},"Help":{},"Deprecated":true,"Type":"String"}`, string(b))
}
