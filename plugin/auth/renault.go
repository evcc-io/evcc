package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/providerauth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/renault/gigya"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("renault", NewRenaultFromConfig)
}

var (
	renaultMu         sync.Mutex
	renaultIdentities = make(map[string]*Renault)
)

type renaultTFA struct {
	regToken       string
	gigyaAssertion string
	phvToken       string
	gmid           string
}

// Renault handles Renault's Gigya email TFA login.
type Renault struct {
	mu       sync.Mutex
	log      *util.Logger
	identity *gigya.Identity
	user     string
	password string
	region   string
	subject  string
	pending  *renaultTFA
	onlineC  chan<- bool
}

// NewRenaultFromConfig creates a Renault auth provider from configuration.
func NewRenaultFromConfig(ctx context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		User, Password, Region string
	}

	cc.Region = "de_DE"

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewRenault(ctx, cc.User, cc.Password, cc.Region)
}

// NewRenault creates a Renault auth provider.
func NewRenault(ctx context.Context, user, password, region string) (*Renault, error) {
	subject := gigya.Subject(user, region)

	renaultMu.Lock()
	defer renaultMu.Unlock()

	if instance := renaultIdentities[subject]; instance != nil {
		instance.user = user
		instance.password = password
		instance.region = region
		return instance, nil
	}

	log := util.ContextLoggerWithDefault(ctx, util.NewLogger(subject)).Redact(user, password)

	renaultKeys := keys.New(log)
	renaultKeys.Load(region)

	r := &Renault{
		log:      log,
		identity: gigya.NewIdentity(log, renaultKeys.Gigya, region),
		user:     user,
		password: password,
		region:   region,
		subject:  subject,
	}

	onlineC, err := providerauth.Register(subject, r)
	if err != nil {
		return nil, err
	}

	r.onlineC = onlineC
	r.onlineC <- r.Authenticated()
	renaultIdentities[subject] = r

	return r, nil
}

var _ api.AuthProvider = (*Renault)(nil)
var _ api.AuthCodeProvider = (*Renault)(nil)
var _ oauth2.TokenSource = (*Renault)(nil)

// Token checks whether Renault login can complete without another TFA code.
func (r *Renault) Token() (*oauth2.Token, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	gmid := gigya.StoredGMID(r.user, r.region)
	res, err := r.identity.LoginRaw(r.user, r.password, gmid)
	if err != nil {
		return nil, err
	}
	if res.ErrorCode == 0 {
		r.notify(true)
		return renaultToken(), nil
	}
	if res.TFARequired() {
		r.notify(false)
		return nil, api.LoginRequiredError(r.subject)
	}

	return nil, res.Error()
}

// Login starts Renault TFA and sends the email verification code.
func (r *Renault) Login(string) (string, *oauth2.DeviceAuthResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	gmid := gigya.StoredGMID(r.user, r.region)
	res, err := r.identity.LoginRaw(r.user, r.password, gmid)
	if err != nil {
		return "", nil, err
	}
	if res.ErrorCode == 0 {
		r.pending = nil
		r.notify(true)
		return "", nil, nil
	}
	if !res.TFARequired() {
		return "", nil, res.Error()
	}
	if res.RegToken == "" {
		return "", nil, errors.New("renault TFA required but missing regToken")
	}

	if gmid == "" {
		gmid = uuid.NewString()
	}

	init, err := r.identity.InitTFA(res.RegToken, gmid)
	if err != nil {
		return "", nil, err
	}
	if init.ErrorCode != 0 {
		return "", nil, init.Error()
	}
	if init.GigyaAssertion == "" {
		return "", nil, errors.New("renault TFA init missing gigyaAssertion")
	}

	send, err := r.identity.SendVerificationCode(init.GigyaAssertion, gmid, r.user)
	if err != nil {
		return "", nil, err
	}
	if send.ErrorCode != 0 {
		return "", nil, send.Error()
	}
	if send.PHVToken == "" {
		return "", nil, errors.New("renault TFA email response missing phvToken")
	}

	r.pending = &renaultTFA{
		regToken:       res.RegToken,
		gigyaAssertion: init.GigyaAssertion,
		phvToken:       send.PHVToken,
		gmid:           gmid,
	}
	r.notify(false)

	return "", nil, nil
}

// CodeInput returns the active Renault verification-code prompt.
func (r *Renault) CodeInput() *api.AuthCodeInput {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pending == nil {
		return nil
	}

	return &api.AuthCodeInput{Message: gigya.VerificationCodeMessage()}
}

// SubmitCode verifies and finalizes the Renault TFA challenge.
func (r *Renault) SubmitCode(code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pending == nil {
		return errors.New("no pending Renault verification code")
	}

	complete, err := r.identity.CompleteVerification(r.pending.gigyaAssertion, r.pending.phvToken, r.pending.gmid, code)
	if err != nil {
		return err
	}
	if complete.ErrorCode != 0 {
		return complete.Error()
	}
	if complete.ProviderAssertion == "" {
		return errors.New("renault TFA verification missing providerAssertion")
	}

	finalize, err := r.identity.FinalizeTFA(r.pending.regToken, r.pending.gigyaAssertion, complete.ProviderAssertion, r.pending.gmid)
	if err != nil {
		return err
	}
	if finalize.ErrorCode != 0 {
		return finalize.Error()
	}

	login, err := r.identity.LoginRaw(r.user, r.password, r.pending.gmid)
	if err != nil {
		return err
	}
	if login.ErrorCode != 0 {
		return fmt.Errorf("renault login after TFA failed: %w", login.Error())
	}

	gigya.StoreGMID(r.user, r.region, r.pending.gmid)
	r.pending = nil
	r.notify(true)

	return nil
}

// Logout removes the trusted Renault device id.
func (r *Renault) Logout() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending = nil
	if err := gigya.DeleteGMID(r.user, r.region); err != nil {
		return err
	}

	r.notify(false)

	return nil
}

// HandleCallback is unused for Renault email TFA.
func (r *Renault) HandleCallback(url.Values) error {
	return errors.New("renault does not use an OAuth callback")
}

// Authenticated returns whether Renault has a trusted device id.
func (r *Renault) Authenticated() bool {
	return gigya.StoredGMID(r.user, r.region) != ""
}

// DisplayName returns the provider name shown in the UI.
func (r *Renault) DisplayName() string {
	return "Renault"
}

func (r *Renault) notify(online bool) {
	if r.onlineC != nil {
		select {
		case r.onlineC <- online:
		default:
		}
	}
}

func renaultToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken: "renault",
		Expiry:      time.Now().Add(time.Hour),
	}
}
