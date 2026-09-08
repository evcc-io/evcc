package charger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.yaml.in/yaml/v4"
)

func TestOpenWB20LoadpointQuery(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	wb := &OpenWB20{}
	other := &OpenWB20{}
	assert.Empty(t, lookupOpenWB20LoadpointQuery(wb))

	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "wallbox"}, api.Charger(wb))))
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "other"}, api.Charger(other))))
	assert.Empty(t, lookupOpenWB20LoadpointQuery(wb), "charger not (yet) referenced by any loadpoint")

	ctrl := gomock.NewController(t)
	lp1 := loadpoint.NewMockAPI(ctrl)
	lp1.EXPECT().GetChargerRef().Return("other").AnyTimes()
	lp2 := loadpoint.NewMockAPI(ctrl)
	lp2.EXPECT().GetChargerRef().Return("wallbox").AnyTimes()
	require.NoError(t, config.Loadpoints().Add(config.NewStaticDevice(config.Named{Name: "lp-1"}, loadpoint.API(lp1))))
	require.NoError(t, config.Loadpoints().Add(config.NewStaticDevice(config.Named{Name: "lp-2"}, loadpoint.API(lp2))))

	assert.Equal(t, "lp=2", lookupOpenWB20LoadpointQuery(wb))
	assert.Equal(t, "lp=1", lookupOpenWB20LoadpointQuery(other))
}

func TestOpenWB20DisplayConfig(t *testing.T) {
	require.Zero(t, network.Config().Port)
	log := util.NewLogger("openwb-2.0")
	writer := log.DEBUG.Writer()
	t.Cleanup(func() { log.DEBUG.SetOutput(writer) })

	for _, test := range []struct {
		name    string
		display any
		message string
	}{
		{"default", nil, "display setup skipped: network not initialized"},
		{"enabled", true, "display setup skipped: network not initialized"},
		{"disabled", false, "display setup disabled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			log.DEBUG.SetOutput(&output)
			config := map[string]any{"uri": "localhost:1502"}
			if test.display != nil {
				config["display"] = test.display
			}
			_, err := NewOpenWB20FromConfig(t.Context(), config)
			require.NoError(t, err)
			assert.Contains(t, output.String(), test.message)
		})
	}
}

func TestOpenWB20DisplayTemplate(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Charger) {
		if tmpl.Template != "openwb-2.0" {
			continue
		}
		for _, enabled := range []bool{true, false} {
			values := tmpl.Defaults(templates.RenderModeUnitTest)
			assert.Equal(t, "true", values["display"])
			values["host"] = "localhost"
			values["display"] = enabled
			values["broker"] = "tls://openwb:8883"
			values["user"] = "display"
			values["password"] = "test-password"
			values[templates.ModbusKeyTCPIP] = true
			tmpl.ModbusValues(templates.RenderModeUnitTest, values)
			data, _, err := tmpl.RenderResult(templates.Charger, templates.RenderModeInstance, values)
			require.NoError(t, err)
			var config map[string]any
			require.NoError(t, yaml.Unmarshal(data, &config))
			assert.Equal(t, enabled, config["display"])
			if enabled {
				assert.Equal(t, values["broker"], config["broker"])
				assert.Equal(t, values["user"], config["user"])
				assert.Equal(t, values["password"], config["password"])
			} else {
				assert.NotContains(t, config, "broker")
			}
		}
		return
	}
	t.Fatal("openwb-2.0 template not found")
}

func TestOpenWB20DisplayDisconnect(t *testing.T) {
	previous := network.Config()
	network.Start(globalconfig.Network{Port: 7070})
	t.Cleanup(func() { network.Start(previous) })

	for _, test := range []struct {
		name      string
		topic     string
		value     string
		cancel    bool
		noSuback  bool
		unchanged bool
		reject    bool
		timeout   bool
		writes    int
	}{
		{name: "configured", writes: 2},
		{name: "primary", topic: "openWB/general/extern", value: "false"},
		{name: "inactive", topic: "openWB/optional/int_display/active", value: "false"},
		{name: "unsupported", topic: "openWB/system/configurable/display_themes", value: "[]"},
		{name: "malformed", topic: "openWB/general/extern", value: "invalid"},
		{name: "canceled", cancel: true},
		{name: "subscription canceled", cancel: true, noSuback: true},
		{name: "unchanged", unchanged: true},
		{name: "missing state", timeout: true},
		{name: "rejected", reject: true, writes: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(10*time.Second)))

			timeout := 10 * time.Second
			if test.timeout || test.reject {
				timeout = time.Second
			}
			ctx, cancel := context.WithTimeout(t.Context(), timeout)
			defer cancel()
			_, err = NewOpenWB20FromConfig(ctx, map[string]any{
				"uri": "localhost:1502", "broker": listener.Addr().String(),
			})
			require.NoError(t, err)
			conn, err := listener.Accept()
			require.NoError(t, err)
			defer conn.Close()
			require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))

			packet, err := packets.ReadPacket(conn)
			require.NoError(t, err)
			connect, ok := packet.(*packets.ConnectPacket)
			require.True(t, ok)
			assert.True(t, connect.CleanSession)
			require.NoError(t, packets.NewControlPacket(packets.Connack).Write(conn))

			values := map[string]string{
				"openWB/general/extern":                     "true",
				"openWB/optional/int_display/active":        "true",
				"openWB/system/configurable/display_themes": `[{"value":"url_display"}]`,
				"openWB/optional/int_display/theme":         `{"type":"cards"}`,
				"openWB/general/extern_display_mode":        `"primary"`,
			}
			if test.topic != "" {
				values[test.topic] = test.value
			}
			if test.unchanged {
				values["openWB/optional/int_display/theme"] = fmt.Sprintf(`{"type":"url_display","configuration":{"url":%q}}`, network.Config().InternalURL())
				values["openWB/general/extern_display_mode"] = `"local"`
			}
			publish := func(topic, payload string) {
				message := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
				message.TopicName = topic
				message.Payload = []byte(payload)
				require.NoError(t, message.Write(conn))
			}

			var writes int
			for {
				packet, err := packets.ReadPacket(conn)
				if err != nil {
					require.ErrorIs(t, err, io.EOF)
					break
				}
				switch packet := packet.(type) {
				case *packets.SubscribePacket:
					ack := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
					ack.MessageID = packet.MessageID
					ack.ReturnCodes = packet.Qoss
					if !test.noSuback {
						require.NoError(t, ack.Write(conn))
					}
					if test.cancel {
						cancel()
						continue
					}
					if test.timeout {
						continue
					}
					for _, topic := range packet.Topics {
						publish(topic, values[topic])
					}
				case *packets.PublishPacket:
					writes++
					assert.False(t, packet.Retain)
					assert.True(t, strings.HasPrefix(packet.TopicName, "openWB/set/"))
					ack := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
					ack.MessageID = packet.MessageID
					require.NoError(t, ack.Write(conn))
					if !test.reject {
						publish(strings.Replace(packet.TopicName, "openWB/set/", "openWB/", 1), string(packet.Payload))
					}
				case *packets.DisconnectPacket:
				default:
					t.Fatalf("unexpected packet: %T", packet)
				}
			}
			assert.Equal(t, test.writes, writes)
		})
	}
}
