package mercedes

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util/request"
	protos "github.com/evcc-io/evcc/vehicle/mercedes/pb"
	"google.golang.org/protobuf/proto"
)

const (
	// vehicles of an account are polled in a row and share one connection
	pushInterval = 10 * time.Second

	// from sdk version 4.5.0 the push api sends a format the generated protobuf does not know
	pushSdkVersion = "4.4.2"
)

// pushHeaders returns the handshake headers of the push api
func pushHeaders(token, region string) http.Header {
	// the push api expects the bare access token without Bearer prefix
	headers := http.Header{"Authorization": {token}}
	for k, val := range mbheaders(false, region) {
		headers.Set(k, val)
	}
	headers.Set("Ris-Sdk-Version", pushSdkVersion)

	return headers
}

// pushUpdates returns the full update of all vehicles of the account from the push api
func (v *Identity) pushUpdates() (map[string]*protos.VEPUpdate, error) {
	token, err := v.Token()
	if err != nil {
		return nil, err
	}

	headers := pushHeaders(token.AccessToken, v.region)

	uri := "wss://websocket." + strings.TrimPrefix(getWidgetUri(v.region), "https://widget.") + "/v2/ws"

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	return pushUpdates(ctx, uri, headers)
}

// pushUpdates reads the full update that the push api sends after connecting
func pushUpdates(ctx context.Context, uri string, headers http.Header) (map[string]*protos.VEPUpdate, error) {
	conn, _, err := websocket.Dial(ctx, uri, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		return nil, err
	}
	defer conn.CloseNow()

	// the full update exceeds the default limit
	conn.SetReadLimit(1 << 20)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err
		}

		var msg protos.PushMessage
		if err := proto.Unmarshal(data, &msg); err != nil {
			return nil, err
		}

		updates := msg.GetVepUpdates()
		if updates == nil {
			continue
		}

		ack, err := proto.Marshal(&protos.ClientMessage{
			Msg: &protos.ClientMessage_AcknowledgeVepUpdatesByVin{
				AcknowledgeVepUpdatesByVin: &protos.AcknowledgeVEPUpdatesByVIN{SequenceNumber: updates.GetSequenceNumber()},
			},
		})
		if err != nil {
			return nil, err
		}

		return updates.GetUpdates(), conn.Write(ctx, websocket.MessageBinary, ack)
	}
}
