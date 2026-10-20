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

// explainCache keeps the last explanation keyed by request, result and language, with the
// payload it was asked for, which goes along with the feedback on it
type explainCache struct {
	key  [sha256.Size]byte
	body explainRequest
	Explanation
}

// optimizerPost posts body to the optimizer service path and decodes the answer into res.
// An error message from the service replaces the transport error.
func (site *Site) optimizerPost(ctx context.Context, path string, body, res any) error {
	req, err := request.New(http.MethodPost, optimizerURI()+path, request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return err
	}

	if sponsor.IsAuthorizedForApi() {
		req.Header.Set("Authorization", "Bearer "+sponsor.Token)
	}

	client := request.NewHelper(site.log)
	client.Timeout = 60 * time.Second

	var raw json.RawMessage
	if err := client.DoJSON(req.WithContext(ctx), &raw); err != nil {
		var msg struct {
			Message string `json:"message"` // optimizer error
		}
		if json.Unmarshal(raw, &msg) == nil && msg.Message != "" {
			return errors.New(msg.Message)
		}
		return err
	}

	return json.Unmarshal(raw, res)
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

	var res Explanation
	if err := site.optimizerPost(ctx, "/optimize/explain", body, &res); err != nil {
		return err
	}

	res.Created, res.Plan = time.Now(), result.Updated
	site.explained = explainCache{key: key, body: body, Explanation: res}
	site.publish("evopt-explanation", res)

	return nil
}

// ExplainFeedback votes on the last explanation: up or down, with an optional comment.
// The plan and the explanation go along, the optimizer service stores nothing before a vote.
func (site *Site) ExplainFeedback(ctx context.Context, vote, comment string) error {
	if vote != "up" && vote != "down" {
		return errors.New("vote must be up or down")
	}

	site.explainMu.Lock()
	defer site.explainMu.Unlock()

	if site.explained.Text == "" {
		return errors.New("nothing explained yet")
	}

	body := struct {
		explainRequest
		Explanation string `json:"explanation"`
		Vote        string `json:"vote"`
		Comment     string `json:"comment,omitempty"`
	}{site.explained.body, site.explained.Text, vote, comment}

	var res struct{}
	return site.optimizerPost(ctx, "/optimize/explain/feedback", body, &res)
}
