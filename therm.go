package main

import (
	"sync"
	"time"

	"git.ekin.gr/go-elec/mcp23s17"
)

/*
MCP23S17 I/O expander config:
GPA: 0x66
  GPA0: Rel-1 OUTPUT
  GPA1: Sens0-1 INPUT
  GPA2: Sens1-1 INPUT
  GPA3: nc OUTPUT
  GPA4: Rel-2 OUTPUT
  GPA5: Sens0-2 INPUT
  GPA6: Sens1-2 INPUT
  GPA7: nc OUTPUT
GPB: 0x06
  GPB0: Rel-3 OUTPUT
  GPB1: Sens0-3 INPUT
  GPB2: Sens1-3 INPUT
  GPB3: nc OUTPUT
  GPB4: nc OUTPUT
  GPB5: nc OUTPUT
  GPB6: nc OUTPUT
  GPB7: nc OUTPUT

Check that SPI interface is enabled on the pi (/boot/config.txt)
*/

// Periodic update of hardware state (on top of interrupts)
const updatePeriod = 5 * time.Second

// Sleep time between state change of relays when several need changed at once
const sleepBetweenRelays = 200 * time.Millisecond

// Minimum time between two SetState calls (milliseconds)
const minSetStatePeriod = 1000

type throttleError struct{}

func (e *throttleError) Error() string {
	return "Throttling: too many requests"
}

type thermState struct {
	Rel1   bool `json:"rel1"`
	Sens01 bool `json:"sens01"`
	Sens11 bool `json:"sens11"`
	Rel2   bool `json:"rel2"`
	Sens02 bool `json:"sens02"`
	Sens12 bool `json:"sens12"`
	Rel3   bool `json:"rel3"`
	Sens03 bool `json:"sens03"`
	Sens13 bool `json:"sens13"`
}

var safeThermState = thermState{Rel1: false, Rel2: false, Rel3: false}

type therm struct {
	mcp        *mcp23s17.Mcp23s17
	state      thermState
	mu         sync.RWMutex
	running    bool
	throttle   sync.Mutex
	lastChange int64
}

func NewTherm(interruptCallback func()) (t *therm, err error) {
	t = &therm{
		lastChange: time.Now().UnixMilli(),
	}

	// Update of stored state when an interrupt is triggered
	if interruptCallback == nil {
		interruptCallback = func() {}
	}
	interrupt := func(val [2]byte, _ error) {
		t.updateState(val)
		interruptCallback()
	}

	t.mcp, err = mcp23s17.NewMcp23s17(
		[2]byte{0x66, 0x06}, // Pins input (1) / output (0) config
		[2]byte{0x00, 0x00}, // Pins default value = all down (0)
		interrupt)
	if err != nil {
		return
	}

	// Periodic update of stored state
	// on top on interrupts, for good measure
	t.running = true
	go func() {
		for t.running {
			val, _ := t.mcp.GetAll()
			t.updateState(val)
			time.Sleep(updatePeriod)
		}
	}()
	return
}

func (t *therm) Close() error {
	t.running = false
	return t.mcp.Close()
}

func (t *therm) updateState(val [2]byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = thermState{
		Rel1:   i2b((val[0] >> 0) & 1),
		Sens01: !i2b((val[0] >> 1) & 1),
		Sens11: !i2b((val[0] >> 2) & 1),
		Rel2:   i2b((val[0] >> 4) & 1),
		Sens02: !i2b((val[0] >> 5) & 1),
		Sens12: !i2b((val[0] >> 6) & 1),
		Rel3:   i2b((val[1] >> 0) & 1),
		Sens03: !i2b((val[1] >> 1) & 1),
		Sens13: !i2b((val[1] >> 2) & 1),
	}
}

func (t *therm) GetState() thermState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

func (t *therm) SetState(s thermState) error {
	// Limiting calls per unit of time
	// Using mutex to throttle concurrent calls
	// eventhough the underlying mcp method is safe.
	t.throttle.Lock()
	defer t.throttle.Unlock()
	now := time.Now().UnixMilli()
	if now-t.lastChange < minSetStatePeriod {
		return &throttleError{}
	}
	t.lastChange = now

	val, err := t.mcp.GetAll()
	if err != nil {
		return err
	}
	v1 := b2i(s.Rel1) << 0
	m1 := byte(0xff ^ (1 << 0))
	v2 := b2i(s.Rel2) << 4
	m2 := byte(0xff ^ (1 << 4))
	v3 := b2i(s.Rel3) << 0
	m3 := byte(0xff ^ (1 << 0))

	// Setting relays one after the other to limit risk
	// of voltage / current surge.
	hasChanged := false

	change := val[0] != (val[0]&m1)|v1
	val[0] = (val[0] & m1) | v1
	if change {
		err = t.mcp.SetAll(val)
		if err != nil {
			return err
		}
		hasChanged = true
	}

	change = val[0] != (val[0]&m2)|v2
	val[0] = (val[0] & m2) | v2
	if change {
		if hasChanged {
			time.Sleep(200 * time.Millisecond)
		}
		err = t.mcp.SetAll(val)
		if err != nil {
			return err
		}
		hasChanged = true
	}

	change = val[1] != (val[1]&m3)|v3
	val[1] = (val[1] & m3) | v3
	if change {
		if hasChanged {
			time.Sleep(200 * time.Millisecond)
		}
		err = t.mcp.SetAll(val)
		if err != nil {
			return err
		}
		hasChanged = true
	}

	t.updateState(val)
	return nil
}

func b2i(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func i2b(i byte) bool {
	return i != 0
}
