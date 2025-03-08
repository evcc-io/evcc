package push

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/go-openapi/runtime"
	"github.com/gotify/go-api-client/v2/client/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreateMessage(params *message.CreateMessageParams, authInfo runtime.ClientAuthInfoWriter) (*message.CreateMessageOK, error) {
	args := m.Called(params, authInfo)
	return args.Get(0).(*message.CreateMessageOK), args.Error(1)
}

func TestNewGotifyFromConfig_ValidConfig(t *testing.T) {
	config := map[string]interface{}{
		"URI":   "http://example.com",
		"Token": "test-token",
	}

	messenger, err := NewGotifyFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, messenger)
}

func TestNewGotifyFromConfig_MissingURI(t *testing.T) {
	config := map[string]interface{}{
		"Token": "test-token",
	}

	messenger, err := NewGotifyFromConfig(config)
	assert.Error(t, err)
	assert.Nil(t, messenger)
}

func TestNewGotifyFromConfig_MissingToken(t *testing.T) {
	config := map[string]interface{}{
		"URI": "http://example.com",
	}

	messenger, err := NewGotifyFromConfig(config)
	assert.Error(t, err)
	assert.Nil(t, messenger)
}

func TestGotify_Send_Success(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("CreateMessage", mock.Anything, mock.Anything).Return(&message.CreateMessageOK{}, nil)

	m := &Gotify{
		log:    util.NewLogger("gotify"),
		uri:    "http://example.com",
		token:  "test-token",
		client: mockClient,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m.uri = server.URL

	m.Send("Test Title", "Test Message")
	mockClient.AssertExpectations(t)
}

func TestGotify_Send_InvalidURL(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("CreateMessage", mock.Anything, mock.Anything).Return(&message.CreateMessageOK{}, nil)

	m := &Gotify{
		log:    util.NewLogger("gotify"),
		uri:    "http://invalid-url",
		token:  "test-token",
		client: mockClient,
	}

	m.Send("Test Title", "Test Message")
}

func TestGotify_Send_Failure(t *testing.T) {
	mockClient := new(MockClient)
	mockClient.On("CreateMessage", mock.Anything, mock.Anything).Return(&message.CreateMessageOK{}, errors.New("failed to send message"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := &Gotify{
		log:    util.NewLogger("gotify"),
		uri:    server.URL,
		token:  "test-token",
		client: mockClient,
	}

	m.Send("Test Title", "Test Message")

	mockClient.AssertExpectations(t)
}
