package easee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util/request"
)

// EnergyObservations retrieves the latest observations needed for energy recovery.
func EnergyObservations(ctx context.Context, helper *request.Helper, charger string) ([]json.RawMessage, error) {
	uri := "https://api.easee.com/state/" + url.PathEscape(charger) + "/observations?ids=109,121,124,129,223"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	var res struct {
		Observations []struct {
			ID        ObservationID   `json:"id"`
			Timestamp time.Time       `json:"timestamp"`
			DataType  DataType        `json:"dataType"`
			Value     json.RawMessage `json:"value"`
		} `json:"observations"`
	}
	if err := helper.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("energy observations: %w", err)
	}
	if len(res.Observations) == 0 {
		return nil, fmt.Errorf("energy observations: empty response")
	}
	observations := make([]json.RawMessage, 0, len(res.Observations))
	for _, item := range res.Observations {
		obs := Observation{ID: item.ID, Timestamp: item.Timestamp, DataType: item.DataType, Value: string(item.Value)}
		if item.DataType == String {
			if err := json.Unmarshal(item.Value, &obs.Value); err != nil {
				return nil, fmt.Errorf("energy observation %s: %w", item.ID, err)
			}
		}
		payload, err := json.Marshal(obs)
		if err != nil {
			return nil, err
		}
		observations = append(observations, payload)
	}
	return observations, nil
}
