package auth

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSetAdminPassword(t *testing.T) {
	t.Skip("skipped until auth is released")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := settings.NewMockAPI(ctrl)
	auth := NewMock(mock)
	password := "testpassword"

	mock.EXPECT().SetString(keys.AdminPassword, gomock.Not(gomock.Eq("")))
	assert.Nil(t, auth.SetAdminPassword(password)) // success
}

func TestRemoveAdminPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := settings.NewMockAPI(ctrl)
	auth := NewMock(mock)

	mock.EXPECT().SetString(keys.JwtSecret, "")
	mock.EXPECT().SetString(keys.AdminPassword, "")
	auth.RemoveAdminPassword()
}

func TestIsAdminPasswordValid(t *testing.T) {
	t.Skip("skipped until auth is released")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := settings.NewMockAPI(ctrl)
	auth := NewMock(mock)

	validPw := "testpassword"
	invalidPw := "wrongpassword"

	// password not set, reject
	mock.EXPECT().String(keys.AdminPassword).Return("", nil).Times(1)
	assert.False(t, auth.IsAdminPasswordValid(validPw))

	// password set, accept
	var storedHash string
	mock.EXPECT().SetString(keys.AdminPassword, gomock.Not(gomock.Eq(""))).Do(func(_ string, hash string) { storedHash = hash })
	auth.SetAdminPassword(validPw)
	mock.EXPECT().String(keys.AdminPassword).Return(storedHash, nil).Times(2)
	assert.True(t, auth.IsAdminPasswordValid(validPw))

	// password set, wrong password
	assert.False(t, auth.IsAdminPasswordValid(invalidPw))
}

func TestJwtToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := settings.NewMockAPI(ctrl)
	auth := NewMock(mock)

	mock.EXPECT().String(keys.JwtSecret).Return("somesecret", nil).AnyTimes()

	lifetime := time.Hour
	tokenString, err := auth.GenerateJwtToken(lifetime)
	assert.Nil(t, err, "token generation failed")
	assert.NotEmpty(t, tokenString, "token is empty")

	ok, err := auth.ValidateJwtToken(tokenString)
	assert.True(t, ok && err == nil, "token is invalid")
}
