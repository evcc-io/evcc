package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCharger is a minimal api.Charger used only for lifetime-context tests.
// It records the context it was created with so tests can assert cancellation.
type fakeCharger struct {
	ctx context.Context
}

func (f *fakeCharger) Status() (api.ChargeStatus, error) { return api.StatusA, nil }
func (f *fakeCharger) Enabled() (bool, error)            { return false, nil }
func (f *fakeCharger) Enable(bool) error                 { return nil }
func (f *fakeCharger) MaxCurrent(int64) error            { return nil }

func setupHandlerTestDB(t *testing.T) {
	t.Helper()
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	config.Reset()
}

func fakeChargerReq() configReq {
	return configReq{
		Properties: config.Properties{Type: "fake"},
		Other:      map[string]any{"marker": "value"},
	}
}

func TestDeviceHandler_DeleteCancelsLifetimeContext(t *testing.T) {
	setupHandlerTestDB(t)

	var captured *fakeCharger
	newFromConf := func(ctx context.Context, _ string, _ map[string]any) (api.Charger, error) {
		captured = &fakeCharger{ctx: ctx}
		return captured, nil
	}

	ctx, cancel, done := startDeviceTimeout()
	conf, err := newDevice(ctx, cancel, templates.Charger, fakeChargerReq(), newFromConf, config.Chargers(), false)
	require.NoError(t, err)
	require.NotNil(t, conf)
	require.NotNil(t, captured)
	close(done) // release the timeout goroutine; cancel ownership moves to the configurable device

	// ctx must still be live after successful creation
	select {
	case <-captured.ctx.Done():
		t.Fatal("ctx unexpectedly cancelled before Delete")
	default:
	}

	require.NoError(t, deleteDevice(conf.ID, config.Chargers()))

	select {
	case <-captured.ctx.Done():
		// expected: Delete propagated the cancel
	case <-time.After(time.Second):
		t.Fatal("ctx not cancelled after Delete")
	}
}

func TestDeviceHandler_UpdateCancelsOldNotNew(t *testing.T) {
	setupHandlerTestDB(t)

	var created []*fakeCharger
	newFromConf := func(ctx context.Context, _ string, _ map[string]any) (api.Charger, error) {
		c := &fakeCharger{ctx: ctx}
		created = append(created, c)
		return c, nil
	}

	// create v1
	ctxNew, cancelNew, doneNew := startDeviceTimeout()
	conf, err := newDevice(ctxNew, cancelNew, templates.Charger, fakeChargerReq(), newFromConf, config.Chargers(), false)
	require.NoError(t, err)
	require.Len(t, created, 1)
	close(doneNew)
	v1 := created[0]

	// update to v2 — use yaml mode to sidestep the template merge path
	// (real templates are only registered for real device types)
	ctxUpd, cancelUpd, doneUpd := startDeviceTimeout()
	updateReq := configReq{
		Properties: config.Properties{Type: "fake"},
		Yaml:       "marker: updated",
		Other:      map[string]any{"marker": "updated"},
	}
	err = updateDevice(ctxUpd, cancelUpd, conf.ID, templates.Charger, updateReq, newFromConf, config.Chargers(), false)
	require.NoError(t, err)
	require.Len(t, created, 2)
	close(doneUpd)
	v2 := created[1]

	// v1 must be cancelled, v2 must stay live
	select {
	case <-v1.ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("old ctx not cancelled after successful Update")
	}

	select {
	case <-v2.ctx.Done():
		t.Fatal("new ctx unexpectedly cancelled after successful Update")
	case <-time.After(50 * time.Millisecond):
		// expected: v2 stays alive
	}

	// cleanup: delete v2 so it too gets cancelled
	require.NoError(t, deleteDevice(conf.ID, config.Chargers()))
	select {
	case <-v2.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("new ctx not cancelled after Delete")
	}
}

func TestDeviceHandler_CreationTimeoutCancelsLifetimeContext(t *testing.T) {
	setupHandlerTestDB(t)

	// override startDeviceTimeout's behaviour by using a short timeout inline
	ctx, cancel := context.WithCancel(context.Background())

	var sawCancellation bool
	var ctxCaptured context.Context
	creatorStarted := make(chan struct{})
	creatorDone := make(chan struct{})

	newFromConf := func(ctx context.Context, _ string, _ map[string]any) (api.Charger, error) {
		ctxCaptured = ctx
		close(creatorStarted)
		<-ctx.Done()
		sawCancellation = true
		close(creatorDone)
		return nil, errors.New("cancelled")
	}

	// simulate creation timeout: cancel the lifetime ctx while newFromConf is blocking
	go func() {
		<-creatorStarted
		cancel()
	}()

	_, err := newDevice(ctx, cancel, templates.Charger, fakeChargerReq(), newFromConf, config.Chargers(), false)
	require.Error(t, err)

	<-creatorDone
	assert.True(t, sawCancellation, "creator must observe ctx cancellation")
	require.NotNil(t, ctxCaptured)
	assert.Error(t, ctxCaptured.Err(), "captured ctx must be cancelled")
}
