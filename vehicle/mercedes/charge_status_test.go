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
		got, err := mapChargeStatus(int32(value))
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for _, value := range []int32{-1, 16, 100} {
		got, err := mapChargeStatus(value)
		require.Error(t, err)
		assert.Equal(t, api.StatusNone, got)
	}
}

func TestChargeStateCache(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr *ChargeStatusAttribute
		want api.ChargeStatus
	}{
		{"missing", nil, api.StatusNone},
		{"no metadata", &ChargeStatusAttribute{Value: 3}, api.StatusNone},
		{"not received", &ChargeStatusAttribute{Metadata: &ChargeStatusMetadata{Status: 1}}, api.StatusNone},
		{"invalid", &ChargeStatusAttribute{Metadata: &ChargeStatusMetadata{Status: 3}}, api.StatusNone},
		{"unavailable", &ChargeStatusAttribute{Metadata: &ChargeStatusMetadata{Status: 4}}, api.StatusNone},
		{"zero means charging", &ChargeStatusAttribute{Metadata: &ChargeStatusMetadata{}}, api.StatusC},
		{"unplugged", &ChargeStatusAttribute{Value: 3, Metadata: &ChargeStatusMetadata{}}, api.StatusA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cache chargeStateCache
			got, err := cache.get("car", func() (map[string]*ChargeStatusAttribute, error) {
				return map[string]*ChargeStatusAttribute{"car": tc.attr}, nil
			})
			assert.Equal(t, tc.want, got)
			if tc.want == api.StatusNone {
				assert.ErrorIs(t, err, api.ErrMustRetry)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChargeStateCacheSharedAndRateLimited(t *testing.T) {
	var cache chargeStateCache
	calls := 0
	fetch := func() (map[string]*ChargeStatusAttribute, error) {
		calls++
		return map[string]*ChargeStatusAttribute{
			"one": {Value: 3, Metadata: &ChargeStatusMetadata{}},
			"two": {Value: 13, Metadata: &ChargeStatusMetadata{}},
		}, nil
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, err := cache.get("one", fetch); assert.NoError(t, err) })
	}
	wg.Wait()
	got, err := cache.get("two", fetch)
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, got)
	assert.Equal(t, 1, calls)

	cache.updated = time.Now().Add(-chargeStatusInterval)
	failure := errors.New("connection failed")
	fail := func() (map[string]*ChargeStatusAttribute, error) { calls++; return nil, failure }
	for range 2 {
		got, err = cache.get("one", fail)
		assert.ErrorIs(t, err, failure)
		assert.Equal(t, api.StatusNone, got)
	}
	assert.Equal(t, 2, calls)
}

func TestReadChargeStates(t *testing.T) {
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
		pushes := []*ChargeStatusPush{
			{CommandUpdates: &Sequence{SequenceNumber: 7}},
			{Updates: &ChargeStatusUpdates{SequenceNumber: 8, Vehicles: map[string]*ChargeStatusUpdate{
				"other": {ChargingStatus: &ChargeStatusAttribute{Value: 3, Metadata: &ChargeStatusMetadata{}}},
			}}},
			{Updates: &ChargeStatusUpdates{SequenceNumber: 9, Vehicles: map[string]*ChargeStatusUpdate{
				"car": {FullUpdate: true, ChargingStatus: &ChargeStatusAttribute{Metadata: &ChargeStatusMetadata{}}},
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
			var ack ChargeStatusAck
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
	states, err := readChargeStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), http.Header{"Authorization": {"test-token"}}, "car")
	require.NoError(t, err)
	assert.Equal(t, int32(3), states["other"].Value)
	assert.Equal(t, int32(0), states["car"].Value)
	assert.NotNil(t, states["car"].Metadata)
	<-done
}

func TestReadChargeStatesMissingAndMalformed(t *testing.T) {
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
			states, err := readChargeStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil, "car")
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Nil(t, states["car"])
			}
		})
	}
}

func TestReadChargeStatesCancellation(t *testing.T) {
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
	_, err := readChargeStates(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil, "car")
	require.Error(t, err)
}

func TestChargeStatusConnection(t *testing.T) {
	for _, region := range []string{"EMEA", "APAC", "NORAM"} {
		uri, headers := chargeStatusConnection("secret", region)
		assert.Equal(t, "secret", headers.Get("Authorization"))
		assert.Equal(t, "PROTO", headers.Get("OUTPUT-FORMAT"))
		assert.NotEmpty(t, headers.Get("APP-SESSION-ID"))
		assert.Equal(t, headers.Get("X-Sessionid"), headers.Get("APP-SESSION-ID"))
		assert.NotEmpty(t, headers.Get("X-Applicationname"))
		assert.Contains(t, uri, "/v2/ws")
	}
}
