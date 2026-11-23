package eebus

import (
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/spine-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type approval struct {
	approve bool
	reason  string
}

// approveRecorder records the decision on each pending configuration write
type approveRecorder struct {
	decisions map[model.MsgCounterType]approval
	durations []time.Duration
}

func newApproveRecorder() *approveRecorder {
	return &approveRecorder{decisions: make(map[model.MsgCounterType]approval)}
}

func (r *approveRecorder) approve(msg model.MsgCounterType, approve bool, reason string) {
	r.decisions[msg] = approval{approve, reason}
}

func (r *approveRecorder) setDuration(d time.Duration, _ bool) error {
	r.durations = append(r.durations, d)
	return nil
}

func failsafeLimitWrite(value float64) []ucapi.PendingDeviceConfiguration {
	return []ucapi.PendingDeviceConfiguration{{
		KeyName: model.DeviceConfigurationKeyNameTypeFailsafeConsumptionActivePowerLimit,
		Value:   &model.DeviceConfigurationKeyValueValueType{ScaledNumber: model.NewScaledNumberType(value)},
	}}
}

func failsafeDurationWrite(d time.Duration) []ucapi.PendingDeviceConfiguration {
	return []ucapi.PendingDeviceConfiguration{{
		KeyName: model.DeviceConfigurationKeyNameTypeFailsafeDurationMinimum,
		Value:   &model.DeviceConfigurationKeyValueValueType{Duration: model.NewDurationType(d)},
	}}
}

// TestDeviceConfiguration_NotControlled covers ATC_COM_PT_CSConnection_002/004 and
// ATC_COM_PT_CSFS_003: failsafe values written in "init" or "failsafe state", i.e.
// before a heartbeat and a following limit, are rejected.
func TestDeviceConfiguration_NotControlled(t *testing.T) {
	c := newTestEEBus(t)
	require.NoError(t, c.run()) // init

	r := newApproveRecorder()
	c.approveDeviceConfigurations(map[model.MsgCounterType][]ucapi.PendingDeviceConfiguration{
		1: failsafeLimitWrite(3500),
		2: failsafeDurationWrite(3 * time.Hour),
	}, r.approve, r.setDuration)

	assert.Equal(t, approval{false, "not controlled by energy guard"}, r.decisions[1])
	assert.Equal(t, approval{false, "not controlled by energy guard"}, r.decisions[2])
	assert.Empty(t, r.durations)
}

// TestDeviceConfiguration_FailsafeDurationRange covers ATC_COM_PT_CSConnection_005/008:
// the Energy Guard chooses the failsafe duration minimum between 2h and 24h.
func TestDeviceConfiguration_FailsafeDurationRange(t *testing.T) {
	c := newTestEEBus(t)
	c.controlled = true

	r := newApproveRecorder()
	c.approveDeviceConfigurations(map[model.MsgCounterType][]ucapi.PendingDeviceConfiguration{
		1: failsafeDurationWrite(time.Hour + 54*time.Minute),
		2: failsafeDurationWrite(2 * time.Hour),
		3: failsafeDurationWrite(3 * time.Hour),
		4: failsafeDurationWrite(24 * time.Hour),
		5: failsafeDurationWrite(25 * time.Hour),
		6: failsafeLimitWrite(3500),
	}, r.approve, r.setDuration)

	assert.Equal(t, approval{false, "failsafe duration below minimum"}, r.decisions[1])
	assert.Equal(t, approval{true, ""}, r.decisions[2])
	assert.Equal(t, approval{true, ""}, r.decisions[3])
	assert.Equal(t, approval{true, ""}, r.decisions[4])
	assert.Equal(t, approval{false, "failsafe duration above maximum"}, r.decisions[5])
	assert.Equal(t, approval{true, ""}, r.decisions[6])

	// [LPC-022/5]: rejecting a duration above our maximum sets the maximum
	assert.Equal(t, []time.Duration{failsafeDurationMax}, r.durations)
	assert.Equal(t, failsafeDurationMax, c.failsafeDuration)
}

func TestFailsafeDurationChecks(t *testing.T) {
	assert.True(t, failsafeDurationTooShort(ucapi.PendingDeviceConfiguration{
		KeyName: model.DeviceConfigurationKeyNameTypeFailsafeDurationMinimum,
	}), "unreadable duration")
	assert.False(t, failsafeDurationTooShort(failsafeLimitWrite(1)[0]), "other key")
	assert.False(t, failsafeDurationTooLong(failsafeLimitWrite(1e9)[0]), "other key")
}
