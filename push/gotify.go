package push

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/evcc-io/evcc/util"
	"github.com/go-openapi/runtime"
	"github.com/gotify/go-api-client/v2/auth"
	"github.com/gotify/go-api-client/v2/client/message"
	"github.com/gotify/go-api-client/v2/gotify"
	"github.com/gotify/go-api-client/v2/models"
)

func init() {
	registry.Add("gotify", NewGotifyFromConfig)
}

type GotifyMessage interface {
	CreateMessage(params *message.CreateMessageParams, authInfo runtime.ClientAuthInfoWriter) (*message.CreateMessageOK, error)
}

type GotifyMessageWrapper struct {
	message *message.Client
}

func (g GotifyMessageWrapper) CreateMessage(params *message.CreateMessageParams, authInfo runtime.ClientAuthInfoWriter) (*message.CreateMessageOK, error) {
	return g.message.CreateMessage(params, authInfo)
}

// Gotify implements the gotify messaging aggregator
type Gotify struct {
	log    *util.Logger
	uri    string
	token  string
	client GotifyMessage
}

// NewGotifyFromConfig creates new Gotify messenger
func NewGotifyFromConfig(other map[string]interface{}) (Messenger, error) {
	var cc struct {
		URI   string
		Token string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}
	if cc.Token == "" {
		return nil, errors.New("missing token")
	}
	log := util.NewLogger("gotify")
	log.Redact(cc.Token)

	gotifyUrl, err := url.Parse(cc.URI)
	if err != nil {
		return nil, errors.New("invalid uri")
	}

	newClient := gotify.NewClient(gotifyUrl, &http.Client{})

	m := &Gotify{
		log:    log,
		uri:    cc.URI,
		token:  cc.Token,
		client: &GotifyMessageWrapper{message: newClient.Message},
	}

	return m, nil
}

// Send sends to all receivers
func (m *Gotify) Send(title, msg string) {
	params := message.NewCreateMessageParams()
	params.Body = &models.MessageExternal{
		Title:   title,
		Message: msg,
	}

	_, err := m.client.CreateMessage(params, auth.TokenAuth(m.token))

	if err != nil {
		m.log.ERROR.Printf("gotify: %v", err)
		return
	}
}
