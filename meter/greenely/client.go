package greenely

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	urlLogin          = "https://api2.greenely.com/v1/login"
	urlFacilitiesBase = "https://api2.greenely.com/v1/facilities/"
	urlCheckAuth      = "https://api2.greenely.com/v1/checkauth"
)

type Client struct {
	// Allow users to set Facility ID to avoid fetching it during login.
	FacilityID int

	c        *http.Client
	email    string
	password string
	headers  map[string]string
}

func NewClient(email, password string) *Client {
	headers := map[string]string{
		"Accept-Language": "sv-SE",
		"User-Agent":      "Android 2 111",
		"Content-Type":    "application/json; charset=utf-8",
	}

	return &Client{
		c:        http.DefaultClient,
		email:    email,
		password: password,
		headers:  headers,
	}
}

func (c *Client) applyHeaders(req *http.Request) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	c.applyHeaders(req)
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (c *Client) post(ctx context.Context, url string, payload any) ([]byte, int, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	c.applyHeaders(req)
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (c *Client) Login(ctx context.Context) error {
	payload := map[string]string{"email": c.email, "password": c.password}
	body, status, err := c.post(ctx, urlLogin, payload)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login failed: status=%d body=%s", status, string(body))
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}
	jwtRaw, ok := m["jwt"].(string)
	if !ok || jwtRaw == "" {
		return fmt.Errorf("login response missing jwt: %s", string(body))
	}

	c.headers["Authorization"] = "JWT " + jwtRaw

	if c.FacilityID == 0 {
		if err := c.getFacilityID(ctx); err != nil {
			return fmt.Errorf("failed to fetch primary facility id: %w", err)
		}
	}
	return nil
}

// CheckAuth checks whether the current JWT is valid. If not, it will attempt
// to re-login and return the final state.
func (c *Client) CheckAuth(ctx context.Context) error {
	_, status, err := c.get(ctx, urlCheckAuth)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}

	return c.Login(ctx)
}

func (c *Client) getFacilityID(ctx context.Context) error {
	body, status, err := c.get(ctx, urlFacilitiesBase)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("failed fetching facilities: status=%d body=%s", status, string(body))
	}
	var resp struct {
		Data []struct {
			ID        int  `json:"id"`
			IsPrimary bool `json:"is_primary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return fmt.Errorf("no facilities returned")
	}
	primaryID := resp.Data[0].ID
	for _, f := range resp.Data {
		if f.IsPrimary {
			primaryID = f.ID
			break
		}
	}

	c.FacilityID = primaryID

	return nil
}

// GetSpotPrice fetches spot-price data. It only considers the date part of the
// 'from' and 'to' parameters, ignoring the time.
func (c *Client) GetSpotPrice(ctx context.Context, from, to time.Time) (SpotPrice, error) {
	start := fmt.Sprintf("?from=%04d-%02d-%02d", from.Year(), from.Month(), from.Day())
	end := fmt.Sprintf("&to=%04d-%02d-%02d", to.Year(), to.Month(), to.Day())
	url := urlFacilitiesBase + fmt.Sprint(c.FacilityID) + "/spot-price" + start + end + "&resolution=hourly"
	fmt.Println("Getting: " + url)
	body, status, err := c.get(ctx, url)
	if err != nil {
		return SpotPrice{}, err
	}
	if status != http.StatusOK {
		return SpotPrice{}, fmt.Errorf("get spot price failed: status=%d body=%s", status, string(body))
	}
	var resp SpotPrice
	if err := json.Unmarshal(body, &resp); err != nil {
		return SpotPrice{}, err
	}

	return resp, nil
}
