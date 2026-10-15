package core

import (
	"context"
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

// ExplainOptimizerResult asks the optimizer service why the last plan looks the way it does
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

	return res.Explanation, nil
}
