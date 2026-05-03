# Easee Expected-Orphan CommandResponse Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Suppress false-positive rogue CommandResponse warnings for sync (HTTP 200) API calls made by evcc, and add the human-readable ObservationID name to the rogue warning message.

**Architecture:** Add an `expectedOrphans map[easee.ObservationID]int` counter to the `Easee` struct. Call sites that POST to 200-returning endpoints pre-register the ObservationID(s) they expect back. The `CommandResponse` handler consumes the counter before deciding whether to emit a rogue WARN.

**Tech Stack:** Go, `charger/easee.go`, `charger/easee_test.go`, `charger/easee/signalr.go` (ObservationID enum — read-only reference).

---

### Task 1: Add `expectedOrphans` field and initialise it

**Files:**
- Modify: `charger/easee.go` (struct definition ~line 67, NewEasee init ~line 117)

**Step 1: Add the field to the struct**

In the `Easee` struct, directly after the `pendingTicks` field (~line 67):

```go
cmdMu         sync.Mutex
pendingTicks   map[int64]chan easee.SignalRCommandResponse
expectedOrphans map[easee.ObservationID]int   // add this line
```

**Step 2: Initialise it in `NewEasee`**

In `NewEasee`, directly after the `pendingTicks` initialisation (~line 117):

```go
pendingTicks:    make(map[int64]chan easee.SignalRCommandResponse),
expectedOrphans: make(map[easee.ObservationID]int),   // add this line
```

**Step 3: Initialise it in `newEasee` test helper**

In `charger/easee_test.go`, in the `newEasee()` helper (~line 36):

```go
pendingTicks:    make(map[int64]chan easee.SignalRCommandResponse),
expectedOrphans: make(map[easee.ObservationID]int),   // add this line
```

**Step 4: Verify it compiles**

```bash
go build ./charger/...
```

Expected: no errors.

**Step 5: Commit**

```bash
git add charger/easee.go charger/easee_test.go
git commit -m "feat(easee): add expectedOrphans map to Easee struct"
```

---

### Task 2: Add `registerExpectedOrphan` and `consumeExpectedOrphan` helpers

**Files:**
- Modify: `charger/easee.go` (after the existing `unregisterPendingTick` helper, ~line 226)

**Step 1: Write the failing test**

In `charger/easee_test.go`, add after `TestEasee_CommandResponse_legitimate`:

```go
func TestEasee_registerAndConsumeExpectedOrphan(t *testing.T) {
    e := newEasee()

    // Not registered yet — consume returns false
    assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))

    // Register once
    e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

    // First consume succeeds
    assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))

    // Second consume fails (counter back to zero)
    assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_registerExpectedOrphan_multipleRegistrations(t *testing.T) {
    e := newEasee()

    // Register twice (two concurrent calls in flight)
    e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)
    e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

    assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
    assert.True(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
    assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}
```

**Step 2: Run the tests to verify they fail**

```bash
go test ./charger/... -run "TestEasee_registerAndConsumeExpectedOrphan|TestEasee_registerExpectedOrphan_multipleRegistrations" -v
```

Expected: FAIL — `e.registerExpectedOrphan undefined`, `e.consumeExpectedOrphan undefined`.

**Step 3: Implement the helpers**

In `charger/easee.go`, add after `unregisterPendingTick`:

```go
func (c *Easee) registerExpectedOrphan(ids ...easee.ObservationID) {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	for _, id := range ids {
		c.expectedOrphans[id]++
	}
}

func (c *Easee) consumeExpectedOrphan(id easee.ObservationID) bool {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	if c.expectedOrphans[id] > 0 {
		c.expectedOrphans[id]--
		return true
	}
	return false
}
```

**Step 4: Run the tests to verify they pass**

```bash
go test ./charger/... -run "TestEasee_registerAndConsumeExpectedOrphan|TestEasee_registerExpectedOrphan_multipleRegistrations" -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add charger/easee.go charger/easee_test.go
git commit -m "feat(easee): add registerExpectedOrphan/consumeExpectedOrphan helpers"
```

---

### Task 3: Update `CommandResponse` handler — check orphan counter and improve rogue message

**Files:**
- Modify: `charger/easee.go` (`CommandResponse` method, ~line 392)

**Step 1: Write the failing test**

In `charger/easee_test.go`, add after the existing `TestEasee_CommandResponse_rogue`:

```go
func TestEasee_CommandResponse_expectedOrphan(t *testing.T) {
    e := newEasee()

    // Pre-register the expected orphan
    e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

    resp := easee.SignalRCommandResponse{
        SerialNumber: "EH123456",
        ID:           int(easee.CIRCUIT_MAX_CURRENT_P1),
        Ticks:        111111111,
        WasAccepted:  true,
        ResultCode:   0,
    }

    raw, err := json.Marshal(resp)
    require.NoError(t, err)

    // Should not panic and should consume the orphan counter
    assert.NotPanics(t, func() {
        e.CommandResponse(raw)
    })

    // Counter should now be zero — a second response would be rogue
    assert.False(t, e.consumeExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1))
}

func TestEasee_CommandResponse_rogueAfterOrphanConsumed(t *testing.T) {
    e := newEasee()

    // Register and immediately consume via CommandResponse
    e.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)

    resp := easee.SignalRCommandResponse{
        SerialNumber: "EH123456",
        ID:           int(easee.CIRCUIT_MAX_CURRENT_P1),
        Ticks:        111111111,
        WasAccepted:  true,
    }
    raw, _ := json.Marshal(resp)
    e.CommandResponse(raw) // consumes the counter

    // A second identical response with counter=0 should be treated as rogue (not panic)
    assert.NotPanics(t, func() {
        e.CommandResponse(raw)
    })

    // pendingTicks untouched
    e.cmdMu.Lock()
    assert.Empty(t, e.pendingTicks)
    e.cmdMu.Unlock()
}
```

**Step 2: Run the tests to verify they fail**

```bash
go test ./charger/... -run "TestEasee_CommandResponse_expectedOrphan|TestEasee_CommandResponse_rogueAfterOrphanConsumed" -v
```

Expected: FAIL — `TestEasee_CommandResponse_expectedOrphan` fails because the current handler does not consume the orphan counter (counter stays at 1 after the call).

**Step 3: Update the `CommandResponse` handler**

Replace the existing `CommandResponse` method body in `charger/easee.go`:

```go
func (c *Easee) CommandResponse(i json.RawMessage) {
	var res easee.SignalRCommandResponse

	if err := json.Unmarshal(i, &res); err != nil {
		c.log.ERROR.Printf("invalid message: %s %v", i, err)
		return
	}

	obsID := easee.ObservationID(res.ID)
	c.log.TRACE.Printf("CommandResponse %s: %+v", res.SerialNumber, res)

	c.cmdMu.Lock()
	ch, ok := c.pendingTicks[res.Ticks]
	c.cmdMu.Unlock()

	if ok {
		ch <- res
		return
	}

	if c.consumeExpectedOrphan(obsID) {
		return
	}

	c.log.WARN.Printf("rogue CommandResponse: charger %s ObservationID=%s Ticks=%d "+
		"(accepted=%v, resultCode=%d) which was not triggered by evcc — "+
		"another system may be controlling this charger",
		res.SerialNumber, obsID, res.Ticks, res.WasAccepted, res.ResultCode)
}
```

Note: `easee.ObservationID(res.ID).String()` is provided automatically by the enumer's `String()` method. For unknown IDs it will fall back to the integer representation — no special handling needed.

**Step 4: Run the tests to verify they pass**

```bash
go test ./charger/... -run "TestEasee_CommandResponse" -v
```

Expected: all four CommandResponse tests PASS (`_rogue`, `_legitimate`, `_expectedOrphan`, `_rogueAfterOrphanConsumed`).

**Step 5: Run the full charger test suite**

```bash
go test ./charger/... -v
```

Expected: all tests PASS.

**Step 6: Commit**

```bash
git add charger/easee.go charger/easee_test.go
git commit -m "feat(easee): handle expected-orphan CommandResponses from sync API calls"
```

---

### Task 4: Register expected orphan at the `Phases1p3p` circuit-level call site

**Files:**
- Modify: `charger/easee.go` (`Phases1p3p` method, ~line 735)

**Step 1: Write the failing test**

In `charger/easee_test.go`, add a new test that exercises the circuit-level `Phases1p3p` path and verifies the orphan counter is pre-registered before the POST:

```go
func TestEasee_Phases1p3p_registersExpectedOrphan(t *testing.T) {
    const siteID = 12345
    const circuitID = 67890
    const chargerID = "TESTTEST"

    e := newEasee()
    e.charger = chargerID
    e.site = siteID
    e.circuit = circuitID

    httpmock.ActivateNonDefault(e.Client)
    defer httpmock.DeactivateAndReset()

    // Mock GET circuit settings
    getURI := fmt.Sprintf("%s/sites/%d/circuits/%d/settings", easee.API, siteID, circuitID)
    maxP1, maxP2, maxP3 := 32.0, 32.0, 32.0
    getResp := easee.CircuitSettings{
        MaxCircuitCurrentP1: &maxP1,
        MaxCircuitCurrentP2: &maxP2,
        MaxCircuitCurrentP3: &maxP3,
    }
    body, _ := json.Marshal(getResp)
    httpmock.RegisterResponder(http.MethodGet, getURI,
        httpmock.NewBytesResponder(200, body))

    // Mock POST circuit settings — return 200 (sync)
    httpmock.RegisterResponder(http.MethodPost, getURI,
        httpmock.NewStringResponder(200, ""))

    err := e.Phases1p3p(1)
    assert.NoError(t, err)

    // After the call, the orphan counter should have been pre-registered
    // and then consumed (if a CommandResponse had arrived). Since no
    // CommandResponse arrived in this test, the counter stays at 1.
    e.cmdMu.Lock()
    count := e.expectedOrphans[easee.CIRCUIT_MAX_CURRENT_P1]
    e.cmdMu.Unlock()
    assert.Equal(t, 1, count, "expected orphan should be registered before the POST")
}
```

**Step 2: Run the test to verify it fails**

```bash
go test ./charger/... -run "TestEasee_Phases1p3p_registersExpectedOrphan" -v
```

Expected: FAIL — counter is 0 (not yet registered).

**Step 3: Add `registerExpectedOrphan` to `Phases1p3p`**

In `charger/easee.go`, in the `Phases1p3p` method, locate the circuit-level branch (the `if c.circuit != 0` block). Add the registration call immediately before `postJSONAndWait`:

```go
// existing code building `data` stays unchanged ...

c.registerExpectedOrphan(easee.CIRCUIT_MAX_CURRENT_P1)
_, err = c.postJSONAndWait(uri, data)
```

**Step 4: Run the test to verify it passes**

```bash
go test ./charger/... -run "TestEasee_Phases1p3p_registersExpectedOrphan" -v
```

Expected: PASS.

**Step 5: Run the full suite**

```bash
go test ./charger/... -v
```

Expected: all tests PASS.

**Step 6: Commit**

```bash
git add charger/easee.go charger/easee_test.go
git commit -m "feat(easee): register expected orphan before circuit settings POST in Phases1p3p"
```

---

### Task 5: Final verification

**Step 1: Run the full charger test suite one more time**

```bash
go test ./charger/... -v
```

Expected: all tests PASS.

**Step 2: Check LSP diagnostics**

Open `charger/easee.go` in the editor and verify no type errors or unused imports appear.

**Step 3: Build the whole project**

```bash
go build ./...
```

Expected: clean build, no errors.
