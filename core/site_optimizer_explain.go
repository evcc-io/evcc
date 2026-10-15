package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

// explainRequest is the optimizer's explain payload: the published plan plus currency and language
type explainRequest struct {
	optimizerResult
	Currency string `json:"currency"`
	Language string `json:"language"`
}

// explainCache keeps the last explanation keyed by request, result and language
type explainCache struct {
	key  [sha256.Size]byte
	text string
}

// ExplainOptimizerResult asks the optimizer service why the last plan looks the way it does.
// Repeated calls for the same plan and language return the previous explanation.
func (site *Site) ExplainOptimizerResult(ctx context.Context, language string) (string, error) {
	site.RLock()
	result := site.optimizerResult
	site.RUnlock()

	if result == nil {
		return "", errors.New("no optimizer result")
	}

	body := explainRequest{optimizerResult: *result, Language: language}
	if site.tariffs != nil {
		body.Currency = site.tariffs.Currency.String()
	}

	keyJSON, err := json.Marshal(struct {
		Req      any    `json:"req"`
		Res      any    `json:"res"`
		Language string `json:"language"`
	}{body.Req, body.Res, language})
	if err != nil {
		return "", err
	}
	key := sha256.Sum256(keyJSON)

	site.explainMu.Lock()
	defer site.explainMu.Unlock()

	if site.explained.key == key {
		return site.explained.text, nil
	}

	req, err := request.New(http.MethodPost, optimizerURI()+"/optimize/explain", request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return "", err
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
			return "", errors.New(res.Message)
		}
		return "", err
	}

	site.explained = explainCache{key: key, text: res.Explanation}

	return res.Explanation, nil
}
