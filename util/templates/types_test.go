package templates

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParamLogic(t *testing.T) {
	require.False(t, (&Param{
		Advanced: true,
		Required: true,
	}).IsAdvanced(), "can't be advanced when required")

	require.False(t, (&Param{
		Deprecated: true,
		Required:   true,
	}).IsRequired(), "can't be required when deprecated")

	b, err := json.Marshal(Param{
		Deprecated: true,
		Advanced:   true, // omitempty
		Required:   true, // omitempty
	})
	require.NoError(t, err)
	require.Equal(t, `{"Name":"","Description":{},"Help":{},"Deprecated":true,"Type":"String"}`, string(b))
}
