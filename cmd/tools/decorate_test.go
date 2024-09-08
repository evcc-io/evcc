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

	enG := func() (float64, error) { return 0, nil }
	phS := func(int) error { return nil }
	phG := func() (int, error) { return 0, nil }
	_ = enG
	_ = phS
	_ = phG

	assert.NotNil(t, decorateTest(c, nil, nil, nil))
	assert.NotNil(t, decorateTest(c, nil, nil, phG))
	assert.NotNil(t, decorateTest(c, nil, phS, nil))
	assert.NotNil(t, decorateTest(c, nil, phS, phG))
	assert.NotNil(t, decorateTest(c, enG, nil, nil))
	assert.NotNil(t, decorateTest(c, enG, nil, phG))
	assert.NotNil(t, decorateTest(c, enG, phS, nil))
	assert.NotNil(t, decorateTest(c, enG, phS, phG))
}
