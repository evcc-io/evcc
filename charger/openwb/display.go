package openwb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/evcc-io/evcc/util"
)

const (
	displaySecondaryTopic = "openWB/general/extern"
	displayActiveTopic    = "openWB/optional/int_display/active"
	displayThemesTopic    = "openWB/system/configurable/display_themes"
	displayThemeTopic     = "openWB/optional/int_display/theme"
	displayModeTopic      = "openWB/general/extern_display_mode"
)

type displayClient interface {
	Listen(string, func(string)) error
	Publish(string, bool, string)
}

type displayTheme struct {
	Type          string `json:"type"`
	Configuration struct {
		URL string `json:"url"`
	} `json:"configuration"`
}

type displayState struct {
	mu      sync.Mutex
	values  map[string]string
	updated chan struct{}
}

func (state *displayState) receive(ctx context.Context, topic string, payload string) {
	if ctx.Err() != nil {
		return
	}
	state.mu.Lock()
	state.values[topic] = payload
	state.mu.Unlock()
	select {
	case state.updated <- struct{}{}:
	default:
	}
}

func (state *displayState) value(topic string) string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.values[topic]
}

func (state *displayState) wait(ctx context.Context, topic string, target any, accept func() bool) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: %w", topic, err)
		}
		if payload := state.value(topic); payload != "" {
			if err := json.Unmarshal([]byte(payload), target); err != nil {
				return fmt.Errorf("%s: %w", topic, err)
			}
			if accept == nil || accept() {
				return nil
			}
		}
		select {
		case <-ctx.Done():
		case <-state.updated:
		}
	}
}

// ConfigureDisplay configures an active secondary display once, using acknowledged MQTT state.
func ConfigureDisplay(ctx context.Context, log *util.Logger, client displayClient, uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.Port() == "0" || parsed.User != nil {
		return fmt.Errorf("invalid internal URL")
	}

	state := &displayState{values: make(map[string]string), updated: make(chan struct{}, 1)}
	subscribe := func(topic string) error {
		return client.Listen(topic, func(payload string) { state.receive(ctx, topic, payload) })
	}
	for _, topic := range []string{displaySecondaryTopic, displayActiveTopic} {
		if err := subscribe(topic); err != nil {
			return err
		}
		var enabled bool
		if err := state.wait(ctx, topic, &enabled, nil); err != nil {
			return err
		}
		if !enabled {
			log.DEBUG.Printf("display setup skipped: %s is not true", topic)
			return nil
		}
	}

	if err := subscribe(displayThemesTopic); err != nil {
		return err
	}
	var themes []struct {
		Value string `json:"value"`
	}
	if err := state.wait(ctx, displayThemesTopic, &themes, nil); err != nil {
		return err
	}
	var supported bool
	for _, theme := range themes {
		supported = supported || theme.Value == "url_display"
	}
	if !supported {
		log.DEBUG.Println("display setup skipped: URL theme not supported")
		return nil
	}

	for _, topic := range []string{displayThemeTopic, displayModeTopic} {
		if err := subscribe(topic); err != nil {
			return err
		}
	}
	var current displayTheme
	if err := state.wait(ctx, displayThemeTopic, &current, nil); err != nil {
		return err
	}
	if current.Type == "" {
		return fmt.Errorf("display theme is missing its type")
	}
	var mode string
	if err := state.wait(ctx, displayModeTopic, &mode, nil); err != nil {
		return err
	}

	publish := func(topic, payload string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, gate := range []string{displaySecondaryTopic, displayActiveTopic} {
			var enabled bool
			if err := json.Unmarshal([]byte(state.value(gate)), &enabled); err != nil || !enabled {
				return fmt.Errorf("display setup stopped: %s is not true", gate)
			}
		}
		log.DEBUG.Printf("configuring display: %s", topic)
		client.Publish(topic, false, payload)
		return nil
	}

	changed := current.Type != "url_display" || current.Configuration.URL != uri
	if changed {
		var desired displayTheme
		desired.Type = "url_display"
		desired.Configuration.URL = uri
		payload, err := json.Marshal(desired)
		if err != nil {
			return err
		}
		if err := publish("openWB/set/optional/int_display/theme", string(payload)); err != nil {
			return err
		}
		if err := state.wait(ctx, displayThemeTopic, &current, func() bool { return current == desired }); err != nil {
			if ctx.Err() != context.Canceled {
				log.WARN.Printf("display theme update not confirmed: %v", err)
			}
			return err
		}
	}
	if mode != "local" {
		if err := publish("openWB/set/general/extern_display_mode", `"local"`); err != nil {
			return err
		}
		if err := state.wait(ctx, displayModeTopic, &mode, func() bool { return mode == "local" }); err != nil {
			if ctx.Err() != context.Canceled {
				log.WARN.Printf("display mode update not confirmed: %v", err)
			}
			return err
		}
		changed = true
	}
	if changed {
		log.INFO.Println("display configured to show evcc")
	} else {
		log.DEBUG.Println("display already configured to show evcc")
	}
	return nil
}
