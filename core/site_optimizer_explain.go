package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// explainRequest is the optimizer's explain payload: the published plan plus currency and language
type explainRequest struct {
	optimizerResult
	Currency string `json:"currency"`
	Language string `json:"language"`
}

// Explanation is a plan explanation, when the optimizer service wrote it and the plan it explains.
// It is published as evopt-explanation, so every client shows the last one.
type Explanation struct {
	Text    string    `json:"explanation"`
	Created time.Time `json:"created"`
	Plan    time.Time `json:"plan"` // updated time of the explained plan
}

var _ api.BytesMarshaler = (*Explanation)(nil)

// MarshalBytes publishes the explanation as one message, like evopt
func (e Explanation) MarshalBytes() ([]byte, error) {
	return json.Marshal(e)
}

// explainCache keeps the last explanation keyed by request, result and language
type explainCache struct {
	key [sha256.Size]byte
	Explanation
}

// ExplainOptimizerResult asks the optimizer service why the last plan looks the way it does.
// Repeated calls for the same plan and language return the previous explanation.
func (site *Site) ExplainOptimizerResult(ctx context.Context, language string) error {
	site.RLock()
	result := site.optimizerResult
	site.RUnlock()

	if result == nil {
		return errors.New("no optimizer result")
	}

	body := explainRequest{optimizerResult: *result, Language: language}
	if site.tariffs != nil {
		body.Currency = site.tariffs.Currency.String()
	}

	keyJSON, err := json.Marshal([]any{body.Req, body.Res, language})
	if err != nil {
		return err
	}
	key := sha256.Sum256(keyJSON)

	site.explainMu.Lock()
	defer site.explainMu.Unlock()

	if site.explained.key == key {
		// a rerun can reproduce the plan, the explanation still fits it
		site.explained.Plan = result.Updated
		site.publish("evopt-explanation", site.explained.Explanation)
		return nil
	}

	req, err := request.New(http.MethodPost, optimizerURI()+"/optimize/explain", request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return err
	}

	if sponsor.IsAuthorizedForApi() {
		req.Header.Set("Authorization", "Bearer "+sponsor.Token)
	}

	client := request.NewHelper(site.log)
	client.Timeout = 60 * time.Second

	var res struct {
		Explanation string `json:"explanation"`
		Message     string `json:"message"` // optimizer error
	}

	if err := client.DoJSON(req.WithContext(ctx), &res); err != nil {
		if res.Message != "" {
			return errors.New(res.Message)
		}
		return err
	}

	site.explained = explainCache{key: key, Explanation: Explanation{
		Text:    res.Explanation,
		Created: time.Now(),
		Plan:    result.Updated,
	}}
	site.publish("evopt-explanation", site.explained.Explanation)

	return nil
}
