package toyota

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSoc(t *testing.T) {
	tests := []struct {
		name    string
		status  func() (Status, error)
		want    float64
		wantErr bool
	}{
		{
			name: "returns battery level in percentage",
			status: func() (Status, error) {
				var s Status
				s.Payload.BatteryLevel = 34
				return s, nil
			},
			want: 34,
		},
		{
			name: "returns error when api.status fails",
			status: func() (Status, error) {
				return Status{}, errors.New("error")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Provider{status: tc.status}

			got, err := p.Soc()

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRange(t *testing.T) {
	tests := []struct {
		name    string
		status  func() (Status, error)
		want    int64
		wantErr string
	}{
		{
			name: "km returns value unchanged",
			status: func() (Status, error) {
				var s Status
				s.Payload.EvRangeWithAc.Value = 150
				s.Payload.EvRangeWithAc.Unit = "km"
				return s, nil
			},
			want: 150,
		},
		{
			name: "km returns value truncated",
			status: func() (Status, error) {
				var s Status
				s.Payload.EvRangeWithAc.Value = 150.9
				s.Payload.EvRangeWithAc.Unit = "km"
				return s, nil
			},
			want: 150,
		},
		{
			name: "mi converts to km and truncates",
			status: func() (Status, error) {
				var s Status
				s.Payload.EvRangeWithAc.Value = 100
				s.Payload.EvRangeWithAc.Unit = "mi"
				return s, nil
			},
			want: 160, // 100 * 1.60934 = 160.934 -> truncated
		},
		{
			name: "unsupported unit returns error",
			status: func() (Status, error) {
				var s Status
				s.Payload.EvRangeWithAc.Value = 1
				s.Payload.EvRangeWithAc.Unit = "f"
				return s, nil
			},
			wantErr: "unsupported unit type: f",
		},
		{
			name: "returns error when status fails",
			status: func() (Status, error) {
				return Status{}, errors.New("error")
			},
			wantErr: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Provider{status: tc.status}

			got, err := p.Range()

			if tc.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
