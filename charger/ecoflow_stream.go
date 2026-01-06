package charger

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://developer-eu.ecoflow.com/us/document/bkw

const (
	ecoflowStreamBaseURL = "https://api-e.ecoflow.com"
	ecoflowStreamAPIPath = "/iot-open/sign/device/quota/all"
)

// EcoflowStream represents an EcoFlow Stream series battery system
type EcoflowStream struct {
	*request.Helper
	log               *util.Logger
	uri               string
	sn                string // main device serial number
	accessKey         string // API access key
	secretKey         string // API secret key for signing
	cache             time.Duration
	cacheTTL          time.Duration
	lastQueryTime     time.Time
	statusG           util.Cacheable[quotaResponse]
	enabled           bool
	maxChargePower    float64
	maxDischargePower float64
	currentPower      float64
	battSoc           float64
}

type quotaResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type quotaAllResponse struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Data    QuotaAllData `json:"data"`
}

type QuotaAllData struct {
	Relay2Onoff                              bool    `json:"relay2Onoff"`         // AC1 switch
	Relay3Onoff                              bool    `json:"relay3Onoff"`         // AC2 switch
	PowGetPvSum                              float64 `json:"powGetPvSum"`         // Real-time PV power (W)
	FeedGridMode                             int     `json:"feedGridMode"`        // Feed-in control (1-off, 2-on)
	GridConnectionPower                      float64 `json:"gridConnectionPower"` // Grid port power (W)
	PowGetSysGrid                            float64 `json:"powGetSysGrid"`       // System real-time grid power (W)
	PowGetSysLoad                            float64 `json:"powGetSysLoad"`       // System real-time load power (W)
	CmsBattSoc                               float64 `json:"cmsBattSoc"`          // Battery SOC (%)
	PowGetBpCms                              float64 `json:"powGetBpCms"`         // Real-time aggregated battery power (W)
	BackupReverseSoc                         int     `json:"backupReverseSoc"`    // Backup reserve level (%)
	CmsMaxChgSoc                             int     `json:"cmsMaxChgSoc"`        // Charge limit (%)
	CmsMinDsgSoc                             int     `json:"cmsMinDsgSoc"`        // Discharge limit (%)
	EnergyStrategyOperateModeSelfPoweredOpen bool    `json:"energyStrategyOperateMode.operateSelfPoweredOpen"`
	EnergyStrategyOperateModeIntelligentOpen bool    `json:"energyStrategyOperateMode.operateIntelligentScheduleModeOpen"`
	QuotaCloudTs                             string  `json:"quota_cloud_ts"`
}

type setQuotaRequest struct {
	SN      string      `json:"sn"`
	CmdId   int         `json:"cmdId"`
	CmdFunc int         `json:"cmdFunc"`
	DirDest int         `json:"dirDest"`
	DirSrc  int         `json:"dirSrc"`
	Dest    int         `json:"dest"`
	NeedAck bool        `json:"needAck"`
	Params  interface{} `json:"params"`
}

type setQuotaResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func init() {
	registry.AddCtx("ecoflow-stream", NewEcoflowStreamFromConfig)
	registry.AddCtx("ecoflow-stream-relay1", NewEcoflowStreamRelay1FromConfig)
	registry.AddCtx("ecoflow-stream-relay2", NewEcoflowStreamRelay2FromConfig)
}

// NewEcoflowStreamFromConfig creates a new EcoFlow Stream device from config
func NewEcoflowStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string        `mapstructure:"uri"`
		SN        string        `mapstructure:"sn"`
		AccessKey string        `mapstructure:"accessKey"`
		SecretKey string        `mapstructure:"secretKey"`
		Cache     time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream: missing uri, sn, accessKey or secretKey")
	}

	return NewEcoflowStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Cache)
}

// NewEcoflowStreamRelay1FromConfig creates AC1 relay (relay2) as a controllable device
func NewEcoflowStreamRelay1FromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string        `mapstructure:"uri"`
		SN        string        `mapstructure:"sn"`
		AccessKey string        `mapstructure:"accessKey"`
		SecretKey string        `mapstructure:"secretKey"`
		Cache     time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream-relay1: missing uri, sn, accessKey or secretKey")
	}

	parent, err := NewEcoflowStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Cache)
	if err != nil {
		return nil, err
	}

	return NewEcoflowStreamRelay(parent.(*EcoflowStream), 2, cc.Cache), nil
}

// NewEcoflowStreamRelay2FromConfig creates AC2 relay (relay3) as a controllable device
func NewEcoflowStreamRelay2FromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string        `mapstructure:"uri"`
		SN        string        `mapstructure:"sn"`
		AccessKey string        `mapstructure:"accessKey"`
		SecretKey string        `mapstructure:"secretKey"`
		Cache     time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream-relay2: missing uri, sn, accessKey or secretKey")
	}

	parent, err := NewEcoflowStream(cc.URI, cc.SN, accessKey, secretKey, cc.Cache)
	if err != nil {
		return nil, err
	}

	return NewEcoflowStreamRelay(parent.(*EcoflowStream), 3, cc.Cache), nil
}

// putJSON sends a PUT request with JSON payload and decodes the response
func putJSON(client *http.Client, url string, data interface{}, res interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(res)
}

// NewEcoflowStream creates a new EcoFlow Stream device
func NewEcoflowStream(uri, sn, accessKey, secretKey string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("ecoflow-stream").Redact(accessKey, secretKey)

	c := &EcoflowStream{
		Helper:    request.NewHelper(log),
		log:       log,
		uri:       strings.TrimSuffix(uri, "/"),
		sn:        sn,
		accessKey: accessKey,
		secretKey: secretKey,
		cache:     cache,
		cacheTTL:  cache,
		enabled:   true,
	}

	// Set authorization header using custom transport with HMAC-SHA256 signature
	c.Client.Transport = &authTransport{
		base:      transport.Default(),
		accessKey: accessKey,
		secretKey: secretKey,
	}

	// Create cached quota fetcher
	c.statusG = util.ResettableCached(c.getQuotaAll, cache)

	// Get main device SN if needed
	if !strings.HasPrefix(sn, "BK") {
		mainSN, err := c.getMainDeviceSN()
		if err != nil {
			return nil, fmt.Errorf("failed to get main device serial number: %w", err)
		}
		c.sn = mainSN
	}

	return c, nil
}

// authTransport adds HMAC-SHA256 signed authentication headers to requests
type authTransport struct {
	base      http.RoundTripper
	accessKey string
	secretKey string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate nonce and timestamp
	nonce := generateNonce()
	timestamp := time.Now().UnixMilli()

	// Build query string or body string for signature
	var signStr string
	if req.URL.RawQuery != "" {
		signStr = req.URL.RawQuery
	} else if req.Body != nil && req.Method != "GET" {
		// For POST/PUT with JSON, body needs to be part of signature
		// Extract from the request - this is handled in getQuotaAll
		if sq := req.Header.Get("X-SignatureData"); sq != "" {
			signStr = sq
			req.Header.Del("X-SignatureData")
		}
	}

	// Add authentication parameters to signature string
	if signStr != "" {
		signStr += "&"
	}
	signStr += fmt.Sprintf("accessKey=%s&nonce=%d&timestamp=%d", t.accessKey, nonce, timestamp)

	// Create HMAC-SHA256 signature
	sign := hmacSHA256(signStr, t.secretKey)

	// Set headers
	req.Header.Set("accessKey", t.accessKey)
	req.Header.Set("nonce", fmt.Sprintf("%d", nonce))
	req.Header.Set("timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("sign", sign)

	return t.base.RoundTrip(req)
}

// generateNonce creates a random 6-digit nonce
func generateNonce() int64 {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return 100000 + n.Int64()
}

// hmacSHA256 creates HMAC-SHA256 signature
func hmacSHA256(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// getMainDeviceSN fetches the main device serial number
func (c *EcoflowStream) getMainDeviceSN() (string, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/system/main/sn?sn=%s", c.uri, c.sn)
	var res quotaResponse
	err := c.GetJSON(uri, &res)
	if err != nil {
		return "", err
	}

	if res.Code != "0" {
		return "", fmt.Errorf("failed to get main serial number: %s", res.Message)
	}

	if data, ok := res.Data["sn"].(string); ok {
		return data, nil
	}

	return "", fmt.Errorf("invalid response format for main SN")
}

// getQuotaAll fetches all quota data from the device
func (c *EcoflowStream) getQuotaAll() (quotaResponse, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", c.uri, c.sn)
	var res quotaAllResponse
	err := c.GetJSON(uri, &res)
	if err != nil {
		return quotaResponse{}, err
	}

	if res.Code != "0" {
		return quotaResponse{}, fmt.Errorf("failed to get quotas: %s", res.Message)
	}

	// Convert to generic response
	data, _ := json.Marshal(res.Data)
	var generic quotaResponse
	generic.Code = res.Code
	generic.Message = res.Message
	json.Unmarshal(data, &generic.Data)

	// Update internal cache
	c.battSoc = res.Data.CmsBattSoc
	c.currentPower = res.Data.PowGetBpCms

	return generic, nil
}

// Status implements the api.Charger interface
func (c *EcoflowStream) Status() (api.ChargeStatus, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	// Determine status based on battery power
	// Positive = charging, Negative = discharging
	if power, ok := data.Data["powGetBpCms"].(float64); ok {
		if power > 100 { // threshold for charging (100W)
			return api.StatusC, nil
		}
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
// Returns true if AC switch is on (relay2Onoff for AC1)
func (c *EcoflowStream) Enabled() (bool, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return false, err
	}

	if relay, ok := data.Data["relay2Onoff"].(bool); ok {
		return relay, nil
	}

	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *EcoflowStream) Enable(enable bool) error {
	return c.setRelay2(enable)
}

// setRelay2 controls the AC1 relay (relay2Onoff)
func (c *EcoflowStream) setRelay2(onoff bool) error {
	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgRelay2Onoff": onoff,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set relay: %s", res.Message)
	}

	c.enabled = onoff
	c.statusG.Reset()
	return nil
}

// MaxCurrent implements the api.Charger interface (not directly supported)
func (c *EcoflowStream) MaxCurrent(current int64) error {
	return fmt.Errorf("not supported")
}

// CurrentPower implements the api.Meter interface
func (c *EcoflowStream) CurrentPower() (float64, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	if power, ok := data.Data["powGetBpCms"].(float64); ok {
		return power, nil
	}

	return 0, nil
}

// TotalEnergy implements the api.MeterEnergy interface
// Note: Stream API doesn't provide direct total energy, would need historical data
func (c *EcoflowStream) TotalEnergy() (float64, error) {
	return 0, fmt.Errorf("not supported")
}

// ChargedEnergy implements the api.ChargeRater interface
func (c *EcoflowStream) ChargedEnergy() (float64, error) {
	return 0, fmt.Errorf("not supported")
}

// Phases implements the api.PhaseSwitcher interface
func (c *EcoflowStream) Phases() (int, error) {
	return 0, fmt.Errorf("not supported")
}

// SetPhases implements the api.PhaseSwitcher interface
func (c *EcoflowStream) SetPhases(phases int) error {
	return fmt.Errorf("not supported")
}

// Soc returns the battery state of charge (implements api.Battery)
func (c *EcoflowStream) Soc() (float64, error) {
	data, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	if soc, ok := data.Data["cmsBattSoc"].(float64); ok {
		return soc, nil
	}

	return 0, nil
}

// SetBackupReserve sets the backup reserve level
func (c *EcoflowStream) SetBackupReserve(level int) error {
	if level < 3 || level > 95 {
		return fmt.Errorf("backup reserve must be between 3 and 95")
	}

	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgBackupReverseSoc": level,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set backup reserve: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

// SetChargingLimit sets the maximum charging level
func (c *EcoflowStream) SetChargingLimit(limit int) error {
	if limit < 50 || limit > 100 {
		return fmt.Errorf("charging limit must be between 50 and 100")
	}

	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgCmsMaxChgSoc": limit,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set charging limit: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

// SetDischargingLimit sets the minimum discharging level
func (c *EcoflowStream) SetDischargingLimit(limit int) error {
	if limit < 0 || limit > 50 {
		return fmt.Errorf("discharging limit must be between 0 and 50")
	}

	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgCmsMinDsgSoc": limit,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set discharging limit: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

// SetFeedinControl sets the feed-in control (1=off, 2=on)
func (c *EcoflowStream) SetFeedinControl(enabled bool) error {
	mode := 1 // off
	if enabled {
		mode = 2 // on
	}

	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgFeedGridMode": mode,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set feed-in control: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

// SetOperatingMode sets the operating mode (self-powered or AI mode)
func (c *EcoflowStream) SetOperatingMode(selfPowered bool) error {
	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgEnergyStrategyOperateMode": map[string]interface{}{
				"operateSelfPoweredOpen":             selfPowered,
				"operateIntelligentScheduleModeOpen": !selfPowered,
			},
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set operating mode: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

// EcoflowStreamRelay represents a relay outlet (AC1 or AC2) as a controllable device
type EcoflowStreamRelay struct {
	*request.Helper
	log      *util.Logger
	parent   *EcoflowStream
	relayNum int // 2 for AC1, 3 for AC2
	cache    time.Duration
	statusG  util.Cacheable[quotaResponse]
}

// NewEcoflowStreamRelay creates a relay outlet as a charger device
func NewEcoflowStreamRelay(parent *EcoflowStream, relayNum int, cache time.Duration) *EcoflowStreamRelay {
	log := util.NewLogger("ecoflow-stream-relay")
	r := &EcoflowStreamRelay{
		Helper:   request.NewHelper(log),
		log:      log,
		parent:   parent,
		relayNum: relayNum,
		cache:    cache,
	}

	r.statusG = util.ResettableCached(r.getQuotaAll, cache)
	return r
}

// getQuotaAll fetches all quota data from parent
func (r *EcoflowStreamRelay) getQuotaAll() (quotaResponse, error) {
	return r.parent.statusG.Get()
}

// Status implements the api.Charger interface
func (r *EcoflowStreamRelay) Status() (api.ChargeStatus, error) {
	return api.StatusA, nil // Always ready, not a charger
}

// Enabled implements the api.Charger interface
func (r *EcoflowStreamRelay) Enabled() (bool, error) {
	data, err := r.statusG.Get()
	if err != nil {
		return false, err
	}

	key := "relay2Onoff"
	if r.relayNum == 3 {
		key = "relay3Onoff"
	}

	if relay, ok := data.Data[key].(bool); ok {
		return relay, nil
	}

	return false, nil
}

// Enable implements the api.Charger interface
func (r *EcoflowStreamRelay) Enable(enable bool) error {
	if r.relayNum == 2 {
		return r.parent.setRelay2(enable)
	}
	return r.parent.setRelay3(enable)
}

// MaxCurrent implements the api.Charger interface
func (r *EcoflowStreamRelay) MaxCurrent(current int64) error {
	return fmt.Errorf("not supported")
}

var _ api.Charger = (*EcoflowStreamRelay)(nil)

// EcoflowStreamPV represents the PV (solar) generation as a meter
type EcoflowStreamPV struct {
	*request.Helper
	log     *util.Logger
	parent  *EcoflowStream
	cache   time.Duration
	statusG util.Cacheable[quotaResponse]
}

// NewEcoflowStreamPV creates a PV meter device
func NewEcoflowStreamPV(parent *EcoflowStream, cache time.Duration) *EcoflowStreamPV {
	log := util.NewLogger("ecoflow-stream-pv")
	p := &EcoflowStreamPV{
		Helper: request.NewHelper(log),
		log:    log,
		parent: parent,
		cache:  cache,
	}

	p.statusG = util.ResettableCached(p.getQuotaAll, cache)
	return p
}

// getQuotaAll fetches all quota data from parent
func (p *EcoflowStreamPV) getQuotaAll() (quotaResponse, error) {
	return p.parent.statusG.Get()
}

// CurrentPower implements the api.Meter interface
func (p *EcoflowStreamPV) CurrentPower() (float64, error) {
	data, err := p.statusG.Get()
	if err != nil {
		return 0, err
	}

	if power, ok := data.Data["powGetPvSum"].(float64); ok {
		return power, nil
	}

	return 0, nil
}

var _ api.Meter = (*EcoflowStreamPV)(nil)

// EcoflowStreamGrid represents the grid connection as a meter
type EcoflowStreamGrid struct {
	*request.Helper
	log     *util.Logger
	parent  *EcoflowStream
	cache   time.Duration
	statusG util.Cacheable[quotaResponse]
}

// NewEcoflowStreamGrid creates a grid meter device
func NewEcoflowStreamGrid(parent *EcoflowStream, cache time.Duration) *EcoflowStreamGrid {
	log := util.NewLogger("ecoflow-stream-grid")
	g := &EcoflowStreamGrid{
		Helper: request.NewHelper(log),
		log:    log,
		parent: parent,
		cache:  cache,
	}

	g.statusG = util.ResettableCached(g.getQuotaAll, cache)
	return g
}

// getQuotaAll fetches all quota data from parent
func (g *EcoflowStreamGrid) getQuotaAll() (quotaResponse, error) {
	return g.parent.statusG.Get()
}

// CurrentPower implements the api.Meter interface
// Positive = consuming from grid, Negative = feeding to grid
func (g *EcoflowStreamGrid) CurrentPower() (float64, error) {
	data, err := g.statusG.Get()
	if err != nil {
		return 0, err
	}

	if power, ok := data.Data["gridConnectionPower"].(float64); ok {
		return power, nil
	}

	return 0, nil
}

var _ api.Meter = (*EcoflowStreamGrid)(nil)

// EcoflowStreamBattery represents the battery as a separate device
type EcoflowStreamBattery struct {
	*request.Helper
	log     *util.Logger
	parent  *EcoflowStream
	cache   time.Duration
	statusG util.Cacheable[quotaResponse]
}

// NewEcoflowStreamBattery creates a battery device
func NewEcoflowStreamBattery(parent *EcoflowStream, cache time.Duration) *EcoflowStreamBattery {
	log := util.NewLogger("ecoflow-stream-battery")
	b := &EcoflowStreamBattery{
		Helper: request.NewHelper(log),
		log:    log,
		parent: parent,
		cache:  cache,
	}

	b.statusG = util.ResettableCached(b.getQuotaAll, cache)
	return b
}

// getQuotaAll fetches all quota data from parent
func (b *EcoflowStreamBattery) getQuotaAll() (quotaResponse, error) {
	return b.parent.statusG.Get()
}

// Soc implements the api.Battery interface
func (b *EcoflowStreamBattery) Soc() (float64, error) {
	data, err := b.statusG.Get()
	if err != nil {
		return 0, err
	}

	if soc, ok := data.Data["cmsBattSoc"].(float64); ok {
		return soc, nil
	}

	return 0, nil
}

// CurrentPower implements the api.Meter interface
// Positive = charging, Negative = discharging
func (b *EcoflowStreamBattery) CurrentPower() (float64, error) {
	data, err := b.statusG.Get()
	if err != nil {
		return 0, err
	}

	if power, ok := data.Data["powGetBpCms"].(float64); ok {
		return power, nil
	}

	return 0, nil
}

var _ api.Battery = (*EcoflowStreamBattery)(nil)
var _ api.Meter = (*EcoflowStreamBattery)(nil)

// setRelay3 controls the AC2 relay (relay3Onoff)
func (c *EcoflowStream) setRelay3(onoff bool) error {
	req := setQuotaRequest{
		SN:      c.sn,
		CmdId:   17,
		CmdFunc: 254,
		DirDest: 1,
		DirSrc:  1,
		Dest:    2,
		NeedAck: true,
		Params: map[string]interface{}{
			"cfgRelay3Onoff": onoff,
		},
	}

	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", c.uri)
	var res setQuotaResponse
	err := putJSON(c.Client, uri, req, &res)
	if err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("failed to set relay 3: %s", res.Message)
	}

	c.statusG.Reset()
	return nil
}

var _ api.Charger = (*EcoflowStream)(nil)
var _ api.Meter = (*EcoflowStream)(nil)
var _ api.Battery = (*EcoflowStream)(nil)
