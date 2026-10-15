package mercedes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestMapChargeStatus(t *testing.T) {
	for value, want := range []api.ChargeStatus{"C", "B", "B", "A", "B", "C", "C", "B", "B", "C", "C", "C", "B", "C", "C", "B"} {
		got, err := mapChargeStatus(int64(value))
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for _, value := range []int64{-1, 16, 100} {
		got, err := mapChargeStatus(value)
		require.Error(t, err)
		assert.Equal(t, api.StatusNone, got)
	}
}

func TestProviderStatus(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr *IntegerAttribute
		want api.ChargeStatus
	}{
		{"missing", nil, api.StatusNone},
		{"no metadata", &IntegerAttribute{Value: 3}, api.StatusNone},
		{"not received", &IntegerAttribute{Metadata: &AttributeMetadata{Status: 1}}, api.StatusNone},
		{"invalid", &IntegerAttribute{Metadata: &AttributeMetadata{Status: 3}}, api.StatusNone},
		{"unavailable", &IntegerAttribute{Metadata: &AttributeMetadata{Status: 4}}, api.StatusNone},
		{"zero means charging", &IntegerAttribute{Metadata: &AttributeMetadata{}}, api.StatusC},
		{"unplugged", &IntegerAttribute{Value: 3, Metadata: &AttributeMetadata{}}, api.StatusA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := Provider{stateG: func() (*VehicleStatusUpdate, error) {
				return &VehicleStatusUpdate{ChargingStatus: tc.attr}, nil
			}}
			got, err := v.Status()
			assert.Equal(t, tc.want, got)
			if tc.want == api.StatusNone {
				assert.ErrorIs(t, err, api.ErrMustRetry)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVehicleStateCacheSharedAndRateLimited(t *testing.T) {
	var cache vehicleStateCache
	calls := 0
	cache.register("one")
	cache.register("two")
	fetch := func(vins []string) (map[string]*VehicleStatusUpdate, error) {
		calls++
		assert.ElementsMatch(t, []string{"one", "two"}, vins)
		return map[string]*VehicleStatusUpdate{
			"one": {ChargingStatus: &IntegerAttribute{Value: 3, Metadata: &AttributeMetadata{}}},
			"two": {ChargingStatus: &IntegerAttribute{Value: 13, Metadata: &AttributeMetadata{}}},
		}, nil
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, err := cache.get("one", fetch); assert.NoError(t, err) })
	}
	wg.Wait()
	got, err := cache.get("two", fetch)
	require.NoError(t, err)
	assert.Equal(t, int64(13), got.GetChargingStatus().GetValue())
	assert.Equal(t, 1, calls)

	cache.updated = time.Now().Add(-vehicleStatusInterval)
	failure := errors.New("connection failed")
	fail := func([]string) (map[string]*VehicleStatusUpdate, error) { calls++; return nil, failure }
	for range 2 {
		got, err = cache.get("one", fail)
		assert.ErrorIs(t, err, failure)
		assert.Nil(t, got)
	}
	assert.Equal(t, 2, calls)
}

func TestReadVehicleStates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		assert.Equal(t, "test-token", r.Header.Get("Authorization"))
		conn, err := websocket.Accept(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.CloseNow()
		if !assert.NoError(t, conn.Write(ctx, websocket.MessageText, []byte("ignore"))) {
			return
		}
		if !assert.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte{0x08, 0x01})) {
			return
		}
		pushes := []*VehicleStatusPush{
			{CommandUpdates: &Sequence{SequenceNumber: 7}},
			{Updates: &VehicleStatusUpdates{SequenceNumber: 6, Vehicles: map[string]*VehicleStatusUpdate{
				"car": {ChargingStatus: &IntegerAttribute{Value: 3, Metadata: &AttributeMetadata{}}},
			}}},
			{Updates: &VehicleStatusUpdates{SequenceNumber: 8, Vehicles: map[string]*VehicleStatusUpdate{
				"other": {FullUpdate: true, ChargingStatus: &IntegerAttribute{Value: 3, Metadata: &AttributeMetadata{}}},
			}}},
			{Updates: &VehicleStatusUpdates{SequenceNumber: 9, Vehicles: map[string]*VehicleStatusUpdate{
				"car": {
					FullUpdate: true, ChargingStatus: &IntegerAttribute{Metadata: &AttributeMetadata{}},
					MaxSoc:                &IntegerAttribute{Value: 80, Metadata: &AttributeMetadata{}},
					Odometer:              &IntegerAttribute{Value: 12408, Unit: 1, Metadata: &AttributeMetadata{}},
					Latitude:              &DoubleAttribute{Value: 52, Metadata: &AttributeMetadata{}},
					Longitude:             &DoubleAttribute{Value: 13, Metadata: &AttributeMetadata{}},
					PreconditioningActive: &BooleanAttribute{Metadata: &AttributeMetadata{}},
				},
			}}},
		}
		for _, push := range pushes {
			data, err := proto.Marshal(push)
			if !assert.NoError(t, err) {
				return
			}
			// A future, unrelated push field must not affect decoding.
			data = protowire.AppendTag(data, 100, protowire.BytesType)
			data = protowire.AppendString(data, "unknown")
			if !assert.NoError(t, conn.Write(ctx, websocket.MessageBinary, data)) {
				return
			}
			_, reply, err := conn.Read(ctx)
			if !assert.NoError(t, err) {
				return
			}
			var ack VehicleStatusAck
			if !assert.NoError(t, proto.Unmarshal(reply, &ack)) {
				return
			}
			if push.CommandUpdates != nil {
				assert.Equal(t, int64(7), ack.GetCommandUpdates().GetSequenceNumber())
			} else {
				assert.Equal(t, push.Updates.SequenceNumber, ack.GetUpdates().GetSequenceNumber())
			}
		}
	}))
	defer server.Close()
	states, err := readVehicleStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), http.Header{"Authorization": {"test-token"}}, "other", "car")
	require.NoError(t, err)
	assert.Equal(t, int64(3), states["other"].ChargingStatus.Value)
	assert.Equal(t, int64(0), states["car"].ChargingStatus.Value)
	assert.NotNil(t, states["car"].ChargingStatus.Metadata)
	v := snapshotProvider(states["car"])
	odo, err := v.Odometer()
	require.NoError(t, err)
	assert.Equal(t, float64(12408), odo)
	limit, err := v.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(80), limit)
	lat, lon, err := v.Position()
	require.NoError(t, err)
	assert.Equal(t, float64(52), lat)
	assert.Equal(t, float64(13), lon)
	active, err := v.Climater()
	require.NoError(t, err)
	assert.False(t, active)
	assert.Empty(t, states["car"].ProtoReflect().GetUnknown())
	<-done
}

func TestReadVehicleStatesMissingAndMalformed(t *testing.T) {
	for _, tc := range []struct {
		name      string
		data      []byte
		wantError bool
	}{
		// Golden wire bytes: push 24 -> updates map 2 -> key "car", full_update 2.
		{"full snapshot without status", []byte{0xc2, 0x01, 0x0b, 0x12, 0x09, 0x0a, 0x03, 'c', 'a', 'r', 0x12, 0x02, 0x10, 0x01}, false},
		{"truncated protobuf", []byte{0xc2, 0x01, 0xff}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := websocket.Accept(w, r, nil)
				if !assert.NoError(t, err) {
					return
				}
				defer conn.CloseNow()
				if !assert.NoError(t, conn.Write(ctx, websocket.MessageBinary, tc.data)) {
					return
				}
				_, _, _ = conn.Read(ctx)
			}))
			defer server.Close()
			states, err := readVehicleStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil, "car", "missing")
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, states["car"])
				assert.Nil(t, states["car"].ChargingStatus)
				assert.Nil(t, states["missing"])
			}
		})
	}
}

func TestReadVehicleStatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.CloseNow()
		cancel()
		_, _, _ = conn.Read(context.Background())
	}))
	defer server.Close()
	_, err := readVehicleStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil, "car")
	require.Error(t, err)
}

func TestVehicleStatusConnection(t *testing.T) {
	for _, region := range []string{"EMEA", "APAC", "NORAM"} {
		uri, headers := vehicleStatusConnection("secret", region)
		assert.Equal(t, "secret", headers.Get("Authorization"))
		assert.Equal(t, "PROTO", headers.Get("OUTPUT-FORMAT"))
		assert.NotEmpty(t, headers.Get("APP-SESSION-ID"))
		assert.Equal(t, headers.Get("X-Sessionid"), headers.Get("APP-SESSION-ID"))
		assert.NotEmpty(t, headers.Get("X-Applicationname"))
		assert.Contains(t, uri, "/v2/ws")
	}
}
