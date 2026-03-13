package charger

// LICENSE: MIT
//
// Protocole Lektrico (découvert par analyse du code source lektricowifi + tests réels) :
//
//   GET  http://<host>/rpc/<Method>   → réponse JSON directe
//   POST http://<host>/rpc            → body JSON-RPC, réponse dans le champ "result"
//
// Endpoint principal : charger_info.get retourne TOUT en une seule requête.
// Pas besoin d'endpoints multiples.
//
// Exemple de réponse charger_info.get :
//   {
//     "charger_state": "B",          ← état IEC brut : A/B/C/D/E/F/B_AUTH/B_PAUSE/OTA/LOCKED
//     "extended_charger_state": "B_AUTH",
//     "session_energy": 38.48,       ← Wh
//     "instant_power": 0.0,          ← W
//     "currents": [0.0, 0.0, 0.0],   ← A, tableau [L1, L2, L3]
//     "voltages": [237.65, 0.0, 0.0],← V, tableau [L1, L2, L3]
//     "total_charged_energy": 9683.844, ← kWh
//     "dynamic_current": 32,         ← courant autorisé actuel (0 = pause, 6-32 = actif)
//     "has_active_errors": false,
//     "charger_is_paused": false,
//     "current_limit_reason": 2,     ← int (0=no_limit, 2=user_limit, ...)
//     "temperature": 18.8,
//     "fw_version": "1.51",
//     "headless": true,              ← true = pas d'auth requise
//     "install_current": 32,
//     ...
//   }

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("lektrico", NewLektricoFromConfig)
}

// États bruts retournés dans charger_state / extended_charger_state
const (
	lektricoIECA          = "A"           // pas de véhicule
	lektricoIECB          = "B"           // véhicule connecté, pas de charge
	lektricoIECBAUTH      = "B_AUTH"      // en attente d'authentification
	lektricoIECBPAUSE     = "B_PAUSE"     // charge en pause (dynamic_current=0)
	lektricoIECBSCHEDULER = "B_SCHEDULER" // pause planifiée
	lektricoIECC          = "C"           // charge active
	lektricoIECD          = "D"           // charge active avec ventilation
	lektricoIECE          = "E"           // erreur
	lektricoIECF          = "F"           // erreur fatale
	lektricoIECOTA        = "OTA"         // mise à jour firmware
	lektricoIECLOCKED     = "LOCKED"      // verrouillé
)

// lektricoInfo correspond exactement à la réponse JSON de charger_info.get
type lektricoInfo struct {
	// État
	ChargerState         string `json:"charger_state"`          // état IEC courant
	ExtendedChargerState string `json:"extended_charger_state"` // état détaillé (B_AUTH, B_PAUSE, etc.)
	ChargerIsPaused      bool   `json:"charger_is_paused"`
	HasActiveErrors      bool   `json:"has_active_errors"`

	// Énergie & puissance
	InstantPower       float64   `json:"instant_power"`        // W
	SessionEnergy      float64   `json:"session_energy"`       // Wh
	TotalChargedEnergy float64   `json:"total_charged_energy"` // kWh
	Currents           []float64 `json:"currents"`             // [L1, L2, L3] en A
	Voltages           []float64 `json:"voltages"`             // [L1, L2, L3] en V

	// Configuration
	DynamicCurrent     int     `json:"dynamic_current"`      // 0=pause, 6-32=courant autorisé
	InstallCurrent     int     `json:"install_current"`      // courant max installé
	CurrentLimitReason int     `json:"current_limit_reason"` // 0=no_limit, 2=user_limit, ...
	Temperature        float64 `json:"temperature"`
	FwVersion          string  `json:"fw_version"`
	Headless           bool    `json:"headless"` // true = pas d'authentification requise
}

// lektricoRPCRequest est le format de la requête POST JSON-RPC
type lektricoRPCRequest struct {
	Src    string                 `json:"src"`
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// lektricoRPCResponse enveloppe la réponse POST (les données sont dans "result")
type lektricoRPCResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Lektrico est l'implémentation api.Charger pour les bornes Lektrico 1P7K / 3P22K
type Lektrico struct {
	log     *util.Logger
	baseURL string
	client  *http.Client
	current int64 // dernier courant valide mémorisé pour Enable()
}

// LektricoConfig est la configuration YAML
type LektricoConfig struct {
	Host string `mapstructure:"host"`
}

// NewLektricoFromConfig crée une instance depuis la configuration EVCC
func NewLektricoFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc LektricoConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Host == "" {
		return nil, fmt.Errorf("lektrico: paramètre 'host' requis (ex: 192.168.1.100)")
	}
	return NewLektrico(cc.Host)
}

// NewLektrico crée la borne et vérifie la connectivité
func NewLektrico(host string) (*Lektrico, error) {
	l := &Lektrico{
		log:     util.NewLogger("lektrico"),
		baseURL: fmt.Sprintf("http://%s/rpc", host),
		client:  &http.Client{Timeout: 10 * time.Second},
		current: 6,
	}

	// Vérification de connectivité via Device_id.Get (réponse minimale, rapide)
	var id struct {
		DeviceID string `json:"device_id"`
	}
	if err := l.get("Device_id.Get", &id); err != nil {
		return nil, fmt.Errorf("lektrico: connexion échouée à %s: %w", host, err)
	}
	l.log.DEBUG.Printf("borne Lektrico connectée: %s (fw %s)", id.DeviceID, l.fwVersion())

	return l, nil
}

// fwVersion lit silencieusement la version firmware pour le log (best-effort)
func (l *Lektrico) fwVersion() string {
	var info lektricoInfo
	if err := l.get("charger_info.get", &info); err != nil {
		return "inconnue"
	}
	return info.FwVersion
}

// ─── Transport HTTP ────────────────────────────────────────────────────────

// get effectue GET http://<host>/rpc/<uri> et décode la réponse JSON directement
func (l *Lektrico) get(uri string, result interface{}) error {
	url := l.baseURL + "/" + uri
	resp, err := l.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", uri, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", uri, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("GET %s lecture: %w", uri, err)
	}

	l.log.TRACE.Printf("GET /%s → %s", uri, string(body))
	return json.Unmarshal(body, result)
}

// post effectue POST http://<host>/rpc avec un body JSON-RPC
// La borne retourne {"id":..., "result":{...}} — on extrait le champ result
func (l *Lektrico) post(method string, params map[string]interface{}) error {
	payload := lektricoRPCRequest{
		Src:    "evcc",
		ID:     rand.Intn(90000000) + 10000000,
		Method: method,
		Params: params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	l.log.TRACE.Printf("POST /rpc %s %v", method, params)

	resp, err := l.client.Post(l.baseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s: HTTP %d", method, resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	l.log.TRACE.Printf("POST /rpc %s → %s", method, string(respBody))

	var rpcResp lektricoRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("POST %s décodage: %w", method, err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("POST %s erreur borne: code=%d msg=%s",
			method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return nil
}

// info lit charger_info.get — une seule requête donne tout
func (l *Lektrico) info() (lektricoInfo, error) {
	var info lektricoInfo
	err := l.get("charger_info.get", &info)
	return info, err
}

// ─── api.Charger (interface obligatoire) ──────────────────────────────────

// Status retourne le statut IEC 61851 (A/B/C/F)
func (l *Lektrico) Status() (api.ChargeStatus, error) {
	info, err := l.info()
	if err != nil {
		return api.StatusNone, err
	}

	if info.HasActiveErrors {
		return api.StatusE, nil
	}

	// On utilise extended_charger_state qui est plus précis que charger_state
	switch info.ExtendedChargerState {
	case lektricoIECA:
		return api.StatusA, nil // pas de véhicule
	case lektricoIECB, lektricoIECBAUTH, lektricoIECBPAUSE,
		lektricoIECBSCHEDULER, lektricoIECLOCKED:
		return api.StatusB, nil // connecté mais pas en charge
	case lektricoIECC, lektricoIECD:
		return api.StatusC, nil // charge active
	case lektricoIECE, lektricoIECF:
		return api.StatusE, nil
	case lektricoIECOTA:
		return api.StatusB, nil // mise à jour en cours → on attend
	default:
		l.log.WARN.Printf("état IEC inconnu: %q", info.ExtendedChargerState)
		return api.StatusA, nil
	}
}

// Enabled retourne true si dynamic_current >= 6 (charge autorisée)
func (l *Lektrico) Enabled() (bool, error) {
	info, err := l.info()
	if err != nil {
		return false, err
	}
	return info.DynamicCurrent >= 6, nil
}

// Enable active ou suspend la charge via dynamic_current
func (l *Lektrico) Enable(enable bool) error {
	var value int64
	if enable {
		value = l.current
		if value < 6 {
			value = 6
		}
		l.log.DEBUG.Printf("Enable → dynamic_current=%dA", value)
	} else {
		value = 0
		l.log.DEBUG.Printf("Disable → dynamic_current=0 (pause)")
	}
	return l.post("dynamic_current.set", map[string]interface{}{
		"dynamic_current": value,
	})
}

// MaxCurrent fixe le courant de charge autorisé (en ampères)
func (l *Lektrico) MaxCurrent(current int64) error {
	if current > 32 {
		current = 32
	}
	var value int64
	if current >= 6 {
		value = current
		l.current = current // mémorisé pour Enable()
	} else {
		value = 0 // en dessous du minimum IEC → pause
	}
	l.log.DEBUG.Printf("MaxCurrent → dynamic_current=%dA", value)
	return l.post("dynamic_current.set", map[string]interface{}{
		"dynamic_current": value,
	})
}

// ─── Interfaces optionnelles ───────────────────────────────────────────────

// CurrentPower implémente api.Meter — puissance instantanée en Watts
func (l *Lektrico) CurrentPower() (float64, error) {
	info, err := l.info()
	return info.InstantPower, err
}

// TotalEnergy implémente api.MeterEnergy — énergie totale en kWh
func (l *Lektrico) TotalEnergy() (float64, error) {
	info, err := l.info()
	return info.TotalChargedEnergy, err
}

// ChargedEnergy implémente api.ChargeRater — énergie de session en kWh
func (l *Lektrico) ChargedEnergy() (float64, error) {
	info, err := l.info()
	return info.SessionEnergy / 1000.0, err // Wh → kWh
}

// Currents implémente api.PhaseCurrents — courants L1, L2, L3 en ampères
func (l *Lektrico) Currents() (float64, float64, float64, error) {
	info, err := l.info()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Currents) < 3 {
		return 0, 0, 0, fmt.Errorf("lektrico: tableau currents incomplet (%d éléments)", len(info.Currents))
	}
	return info.Currents[0], info.Currents[1], info.Currents[2], nil
}

// Voltages implémente api.PhaseVoltages — tensions L1, L2, L3 en volts
func (l *Lektrico) Voltages() (float64, float64, float64, error) {
	info, err := l.info()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Voltages) < 3 {
		return 0, 0, 0, fmt.Errorf("lektrico: tableau voltages incomplet (%d éléments)", len(info.Voltages))
	}
	return info.Voltages[0], info.Voltages[1], info.Voltages[2], nil
}
