package mercedes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPushUpdates(t *testing.T) {
	acked := make(chan int32, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(acked)

		conn, err := websocket.Accept(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.CloseNow()

		assert.Equal(t, "token", r.Header.Get("Authorization"))

		for _, msg := range []*protos.PushMessage{
			{Msg: &protos.PushMessage_ApptwinPendingCommandRequest{
				ApptwinPendingCommandRequest: &protos.AppTwinPendingCommandsRequest{},
			}},
			{Msg: &protos.PushMessage_VepUpdates{VepUpdates: &protos.VEPUpdatesByVIN{
				SequenceNumber: 7,
				Updates:        map[string]*protos.VEPUpdate{"vin": {FullUpdate: true}},
			}}},
		} {
			data, err := proto.Marshal(msg)
			assert.NoError(t, err)
			assert.NoError(t, conn.Write(r.Context(), websocket.MessageBinary, data))
		}

		_, data, err := conn.Read(r.Context())
		if !assert.NoError(t, err) {
			return
		}

		var ack protos.ClientMessage
		assert.NoError(t, proto.Unmarshal(data, &ack))
		acked <- ack.GetAcknowledgeVepUpdatesByVin().GetSequenceNumber()
	}))
	defer srv.Close()

	uri := "ws" + strings.TrimPrefix(srv.URL, "http")
	updates, err := pushUpdates(t.Context(), uri, http.Header{"Authorization": {"token"}})
	require.NoError(t, err)
	assert.True(t, updates["vin"].GetFullUpdate())
	assert.Equal(t, int32(7), <-acked)
}

func TestPushHeaders(t *testing.T) {
	for _, region := range []string{"EMEA", "APAC", "NORAM"} {
		t.Run(region, func(t *testing.T) {
			headers := pushHeaders("token", region)

			assert.Equal(t, "token", headers.Get("Authorization"))
			assert.Equal(t, pushSdkVersion, headers.Get("Ris-Sdk-Version"))
			assert.NotEmpty(t, headers.Get("X-Applicationname"))
		})
	}
}

func TestPushAttributes(t *testing.T) {
	intValue := func(value int64) *protos.VehicleAttributeStatus {
		return &protos.VehicleAttributeStatus{AttributeType: &protos.VehicleAttributeStatus_IntValue{IntValue: value}}
	}
	nilValue := &protos.VehicleAttributeStatus{AttributeType: &protos.VehicleAttributeStatus_NilValue{NilValue: true}}
	boolValue := func(value bool) *protos.VehicleAttributeStatus {
		return &protos.VehicleAttributeStatus{AttributeType: &protos.VehicleAttributeStatus_BoolValue{BoolValue: value}}
	}
	unsupported := &protos.VehicleAttributeStatus{AttributeType: &protos.VehicleAttributeStatus_UnsupportedValue{}}
	miles := intValue(1000)
	miles.DisplayUnit = &protos.VehicleAttributeStatus_DistanceUnit_{DistanceUnit: protos.VehicleAttributeStatus_MILES}

	tc := []struct {
		name       string
		attributes map[string]*protos.VehicleAttributeStatus
		odometer   float64
		odoErr     error
		status     api.ChargeStatus
		statusErr  error
		limitSoc   int64
		limitErr   error
		climater   bool
	}{
		{"charging", map[string]*protos.VehicleAttributeStatus{"odo": intValue(2188), "chargingstatus": intValue(0), "maxSoc": intValue(80), "precondNow": boolValue(true), "precondActive": boolValue(false)}, 2188, nil, api.StatusC, nil, 80, nil, true},
		{"plugged", map[string]*protos.VehicleAttributeStatus{"odo": miles, "chargingstatus": intValue(8), "maxSoc": intValue(100), "precondActive": boolValue(true)}, 1609.344, nil, api.StatusB, nil, 100, nil, true},
		{"unplugged", map[string]*protos.VehicleAttributeStatus{"odo": nilValue, "chargingstatus": intValue(3), "maxSoc": nilValue, "precondNow": boolValue(false)}, 0, api.ErrNotAvailable, api.StatusA, nil, 0, api.ErrNotAvailable, false},
		{"nil status", map[string]*protos.VehicleAttributeStatus{"chargingstatus": nilValue}, 0, api.ErrNotAvailable, api.StatusNone, api.ErrNotAvailable, 0, api.ErrNotAvailable, false},
		{"unsupported", map[string]*protos.VehicleAttributeStatus{"odo": unsupported, "chargingstatus": unsupported, "maxSoc": unsupported}, 0, api.ErrNotAvailable, api.StatusNone, api.ErrNotAvailable, 0, api.ErrNotAvailable, false},
		{"untyped", map[string]*protos.VehicleAttributeStatus{"odo": {}, "chargingstatus": {}, "maxSoc": {}}, 0, api.ErrNotAvailable, api.StatusNone, api.ErrNotAvailable, 0, api.ErrNotAvailable, false},
		{"missing", nil, 0, api.ErrNotAvailable, api.StatusNone, api.ErrNotAvailable, 0, api.ErrNotAvailable, false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			v := &Provider{pushG: func() (*protos.VEPUpdate, error) {
				return &protos.VEPUpdate{Attributes: tc.attributes}, nil
			}}

			odometer, err := v.Odometer()
			require.ErrorIs(t, err, tc.odoErr)
			assert.InDelta(t, tc.odometer, odometer, 1e-6)

			status, err := v.Status()
			require.ErrorIs(t, err, tc.statusErr)
			assert.Equal(t, tc.status, status)

			limitSoc, err := v.GetLimitSoc()
			require.ErrorIs(t, err, tc.limitErr)
			assert.Equal(t, tc.limitSoc, limitSoc)

			climater, err := v.Climater()
			require.NoError(t, err)
			assert.Equal(t, tc.climater, climater)
		})
	}
}
