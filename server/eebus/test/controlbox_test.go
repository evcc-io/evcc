package eebus

import (
	"sync"
	"testing"

	"github.com/enbility/eebus-go/usecases/eg/lpc"
	"github.com/enbility/eebus-go/usecases/eg/lpp"
	shipapi "github.com/enbility/ship-go/api"
	"github.com/stretchr/testify/assert"
)

func TestControlboxConnectionEvents(t *testing.T) {
	var box controlbox
	var identity shipapi.ServiceIdentity
	var wg sync.WaitGroup

	wg.Go(func() {
		for range 1000 {
			box.RemoteServiceConnected(nil, identity)
			box.RemoteServiceDisconnected(nil, identity)
		}
	})
	wg.Go(func() {
		for range 1000 {
			box.OnLPCEvent("", nil, nil, lpc.DataUpdateLimit)
			box.OnLPPEvent("", nil, nil, lpp.UseCaseSupportUpdate)
		}
	})
	wg.Wait()

	before := len(box.remoteEntity(lpc.DataUpdateLimit))
	box.RemoteServiceConnected(nil, identity)
	box.OnLPCEvent("", nil, nil, lpc.DataUpdateLimit)
	assert.Len(t, box.remoteEntity(lpc.DataUpdateLimit), before+1)
	box.RemoteServiceDisconnected(nil, identity)
	box.OnLPCEvent("", nil, nil, lpc.DataUpdateLimit)
	assert.Len(t, box.remoteEntity(lpc.DataUpdateLimit), before+1)
}
