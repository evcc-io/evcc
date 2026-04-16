package connectedcars

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

// API provides access to the Connected Cars GraphQL API.
type API struct {
	*request.Helper
	authHelper *request.Helper // plain client for token refresh; no oauth2 transport to avoid circular dependency
	domain     string
	namespace  string
}

// graphqlRequest is a generic GraphQL request body.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// NewAPI creates a new Connected Cars API client with device-token authentication.
func NewAPI(log *util.Logger, domain, namespace, deviceToken string) *API {
	api := &API{
		Helper:     request.NewHelper(log),
		authHelper: request.NewHelper(log),
		domain:     domain,
		namespace:  namespace,
	}

	// Use RefreshTokenSource to handle JWT refresh via device token.
	// The device token is stored as RefreshToken; it never changes.
	token := &oauth2.Token{
		RefreshToken: deviceToken,
	}

	ts := oauth.RefreshTokenSource(token, api.refreshToken)

	// Install oauth2.Transport for Bearer token, plus a decorator for the
	// namespace header required by all API endpoints.
	api.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"X-Organization-Namespace": namespace,
		}),
		Base: &oauth2.Transport{
			Source: ts,
			Base:   api.Client.Transport,
		},
	}

	return api
}

// refreshToken exchanges the device token for a new JWT access token.
func (a *API) refreshToken(token *oauth2.Token) (*oauth2.Token, error) {
	data := struct {
		DeviceToken string `json:"deviceToken"`
	}{
		DeviceToken: token.RefreshToken,
	}

	uri := fmt.Sprintf("https://auth-api.%s/auth/login/deviceToken", a.domain)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Content-Type":             request.JSONContent,
		"X-Organization-Namespace": a.namespace,
	})
	if err != nil {
		return nil, err
	}

	var res TokenResponse

	// Use the plain authHelper (no oauth2 transport) to avoid circular dependency.
	if err := a.authHelper.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("device token login: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  res.Token,
		RefreshToken: token.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(res.Expires) * time.Second),
	}, nil
}

// Vehicles returns the list of vehicles on the account.
func (a *API) Vehicles() ([]Vehicle, error) {
	uri := fmt.Sprintf("https://api.%s/graphql", a.domain)

	body := graphqlRequest{
		Query: `{ vehicles(first:100) { items { id licensePlate vin } } }`,
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return nil, err
	}

	var res VehiclesResponse
	if err := a.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}

	if res.Data == nil {
		return nil, fmt.Errorf("list vehicles: missing data in response")
	}

	return res.Data.Vehicles.Items, nil
}

// Data fetches the current vehicle telemetry data.
func (a *API) Data(vehicleID string) (VehicleData, error) {
	uri := fmt.Sprintf("https://api.%s/graphql", a.domain)

	body := graphqlRequest{
		Query:     `query($id: ID!) { vehicle(id: $id) { id chargePercentage { pct } odometer { odometer } rangeTotalKm { km } }}`,
		Variables: map[string]any{"id": vehicleID},
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(body), request.JSONEncoding)
	if err != nil {
		return VehicleData{}, err
	}

	var res DataResponse
	if err := a.DoJSON(req, &res); err != nil {
		return VehicleData{}, fmt.Errorf("vehicle data: %w", err)
	}

	if res.Data == nil {
		return VehicleData{}, fmt.Errorf("vehicle data: missing data in response")
	}

	return res.Data.Vehicle, nil
}
