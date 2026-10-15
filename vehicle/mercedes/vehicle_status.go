package mercedes

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"google.golang.org/protobuf/proto"
)

const vehicleStatusInterval = time.Minute

type vehicleStateCache struct {
	sync.Mutex
	updated time.Time
	states  map[string]*VehicleStatusUpdate
	err     error
}

func (c *vehicleStateCache) get(vin string, fetch func() (map[string]*VehicleStatusUpdate, error)) (*VehicleStatusUpdate, error) {
	c.Lock()
	defer c.Unlock()

	// Account-wide, including failures; detection cache resets must not cause reconnect storms.
	if time.Since(c.updated) >= vehicleStatusInterval {
		c.states, c.err = fetch()
		c.updated = time.Now()
	}
	if c.err != nil {
		return nil, c.err
	}
	state := c.states[vin]
	if state == nil {
		return nil, api.ErrMustRetry
	}
	return state, nil
}

func (v *Identity) vehicleStatus(vin string) (*VehicleStatusUpdate, error) {
	return v.vehicleState.get(vin, func() (map[string]*VehicleStatusUpdate, error) {
		token, err := v.Token()
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		uri, headers := vehicleStatusConnection(token.AccessToken, v.region)
		return readVehicleStates(ctx, uri, headers, vin)
	})
}

func vehicleStatusConnection(token, region string) (string, http.Header) {
	headers := make(http.Header)
	for k, v := range mbheaders(false, region) {
		headers.Set(k, v)
	}
	headers.Set("Authorization", token)
	headers.Set("APP-SESSION-ID", headers.Get("X-Sessionid"))
	headers.Set("OUTPUT-FORMAT", "PROTO")
	headers.Set("Ris-Sdk-Version", "4.10.0")
	headers.Set("Ris-Application-Version", "1.68.0 (3060)")
	headers.Set("User-Agent", "Mercedes-Benz/3044 CFNetwork/3860.400.22 Darwin/25.3.0")
	host := "emea"
	switch region {
	case "APAC":
		host = "amap"
		headers.Set("Ris-Application-Version", "1.67.0")
		headers.Set("User-Agent", "mycar-store-ap 1.67.0, ios 26.3, SDK 4.10.0")
	case "NORAM":
		host = "amap"
		headers.Set("Ris-Application-Version", "3.67.0")
		headers.Set("User-Agent", "mycar-store-us v3.67.0, ios 26.3, SDK 4.10.0")
	}
	return "wss://websocket." + host + "-prod.mobilesdk.mercedes-benz.com/v2/ws", headers
}

// readVehicleStates consumes the initial snapshot and closes the connection instead of running a background stream.
func readVehicleStates(ctx context.Context, uri string, headers http.Header, vin string) (map[string]*VehicleStatusUpdate, error) {
	conn, _, err := websocket.Dial(ctx, uri, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		return nil, err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(8 << 20)

	states := make(map[string]*VehicleStatusUpdate)
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		if typ != websocket.MessageBinary {
			continue
		}
		var push VehicleStatusPush
		if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &push); err != nil {
			return nil, err
		}
		var ack VehicleStatusAck
		if push.CommandUpdates != nil {
			ack.CommandUpdates = push.CommandUpdates
		}
		ready := false
		if updates := push.Updates; updates != nil {
			ack.Updates = &Sequence{SequenceNumber: updates.SequenceNumber}
			for name, update := range updates.Vehicles {
				if update.FullUpdate {
					states[name] = update
					ready = ready || name == vin
				}
			}
		}
		if ack.Updates != nil || ack.CommandUpdates != nil {
			data, err := proto.Marshal(&ack)
			if err != nil {
				return nil, err
			}
			if err := conn.Write(ctx, websocket.MessageBinary, data); err != nil {
				return nil, err
			}
		}
		if ready {
			return states, nil
		}
	}
}

func mapChargeStatus(value int64) (api.ChargeStatus, error) {
	switch value {
	case 0, 5, 6, 9, 10, 11, 13, 14:
		return api.StatusC, nil
	case 1, 2, 4, 7, 8, 12, 15:
		return api.StatusB, nil
	case 3:
		return api.StatusA, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown Mercedes charging status: %d", value)
	}
}
