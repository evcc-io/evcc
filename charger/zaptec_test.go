package charger

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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

type zaptecRoundTripper func(*http.Request) (*http.Response, error)

func (f zaptecRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestZaptecEnableRejection(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"accepted", http.StatusOK, "", false},
		{"rejected by Go 2", http.StatusInternalServerError, `{"Code":528,"Details":"Charging is not Paused nor Scheduled; Resume command cannot be sent"}`, false},
		{"rejected by Pro", http.StatusInternalServerError, `{"Code":520,"Details":null,"StackTrace":null}`, false},
		{"failed", http.StatusInternalServerError, `{"Code":500}`, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			helper := request.NewHelper(util.NewLogger("foo"))
			helper.Transport = zaptecRoundTripper(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "/api/chargers/id/sendCommand/507", req.URL.Path)
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})

			c := &Zaptec{
				Helper:   helper,
				instance: zaptec.Charger{Id: "id"},
				statusG: util.ResettableCached(func() (zaptec.StateResponse, error) {
					return nil, nil
				}, 0),
			}

			err := c.Enable(true)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, !tc.wantErr, c.enabled)
		})
	}
}
