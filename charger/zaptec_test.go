package charger

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZaptecDetectVersion(t *testing.T) {
	tests := []struct {
		name    string
		state   zaptec.StateResponse
		err     error
		want    int
		wantErr bool
	}{
		{
			name:  "Go",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Go"}`}},
			want:  zaptec.ZaptecGo,
		},
		{
			name:  "Go2",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Go","ProductVariant":"Go2"}`}},
			want:  zaptec.ZaptecGo2,
		},
		{
			name:  "Go2 takes precedence over Pro",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Pro","ProductVariant":"Go2"}`}},
			want:  zaptec.ZaptecGo2,
		},
		{
			name:  "Pro without device type",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"ProductVariant":"ProMID"}`}},
			want:  zaptec.ZaptecPro,
		},
		{
			name:  "Pro with device type",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Pro","ProductVariant":"ProMID"}`}},
			want:  zaptec.ZaptecPro,
		},
		{
			name:  "Pro without product variant",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Pro"}`}},
			want:  zaptec.ZaptecPro,
		},
		{
			name:  "unknown model",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"DeviceType":"Unknown","ProductVariant":"Unknown"}`}},
			want:  zaptec.ZaptecGo,
		},
		{
			name: "missing capabilities",
			want: zaptec.ZaptecGo,
		},
		{
			name:  "empty capabilities",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities}},
			want:  zaptec.ZaptecGo,
		},
		{
			name:  "missing model fields",
			state: zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{}`}},
			want:  zaptec.ZaptecGo,
		},
		{
			name:    "invalid capabilities",
			state:   zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{`}},
			wantErr: true,
		},
		{
			name:    "state error",
			err:     errors.New("state unavailable"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Zaptec{
				statusG: util.ResettableCached(func() (zaptec.StateResponse, error) {
					return tt.state, tt.err
				}, 0),
			}

			got, err := c.detectVersion()
			if tt.wantErr {
				require.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestZaptecConnectionDurationIgnoresEmptySession(t *testing.T) {
	var state zaptec.StateResponse
	c := &Zaptec{
		statusG: util.ResettableCached(func() (zaptec.StateResponse, error) {
			return state, nil
		}, 0),
	}

	// no session seen yet
	d, err := c.ConnectionDuration()
	require.NoError(t, err)
	assert.Zero(t, d)

	state = zaptec.StateResponse{{StateId: zaptec.SessionIdentifier, ValueAsString: "a"}}
	_, err = c.ConnectionDuration()
	require.NoError(t, err)
	start := c.sessionStart
	assert.False(t, start.IsZero())

	// observation missing from state response: keep session
	state = zaptec.StateResponse{}
	_, err = c.ConnectionDuration()
	require.NoError(t, err)
	assert.Equal(t, "a", c.session)
	assert.Equal(t, start, c.sessionStart)

	// empty identifier: keep session
	state = zaptec.StateResponse{{StateId: zaptec.SessionIdentifier}}
	_, err = c.ConnectionDuration()
	require.NoError(t, err)
	assert.Equal(t, "a", c.session)
	assert.Equal(t, start, c.sessionStart)

	// new identifier: restart
	state = zaptec.StateResponse{{StateId: zaptec.SessionIdentifier, ValueAsString: "b"}}
	_, err = c.ConnectionDuration()
	require.NoError(t, err)
	assert.Equal(t, "b", c.session)
	assert.NotEqual(t, start, c.sessionStart)
}
