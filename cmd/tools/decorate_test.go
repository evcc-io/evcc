package main

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDecorator(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := api.NewMockCharger(ctrl)
	assert.NotNil(t, decorateTest(c, nil))
	assert.NotNil(t, decorateTest(c, func() (float64, error) { return 0, nil }))
}
