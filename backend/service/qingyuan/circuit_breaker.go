package qingyuan

import (
	"sync"
	"time"
)

const (
	CircuitClosed   = "closed"
	CircuitOpen     = "open"
	CircuitHalfOpen = "half_open"
)

type circuitState struct {
	state       string
	failures    int
	openedUntil time.Time
}

var circuitStore = struct {
	mu     sync.Mutex
	states map[int]*circuitState
}{states: map[int]*circuitState{}}

func GetCircuitState(policy ResolvedPolicy) string {
	if policy.Policy == nil || !policy.Config.CircuitBreaker.Enabled {
		return CircuitClosed
	}
	id := policy.Policy.Id
	circuitStore.mu.Lock()
	defer circuitStore.mu.Unlock()
	st := getCircuitStateLocked(id)
	if st.state == CircuitOpen && time.Now().After(st.openedUntil) {
		st.state = CircuitHalfOpen
	}
	return st.state
}

func RecordCircuitSuccess(policy ResolvedPolicy) {
	if policy.Policy == nil || !policy.Config.CircuitBreaker.Enabled {
		return
	}
	circuitStore.mu.Lock()
	defer circuitStore.mu.Unlock()
	st := getCircuitStateLocked(policy.Policy.Id)
	st.state = CircuitClosed
	st.failures = 0
	st.openedUntil = time.Time{}
}

func RecordCircuitFailure(policy ResolvedPolicy) string {
	if policy.Policy == nil || !policy.Config.CircuitBreaker.Enabled {
		return CircuitClosed
	}
	circuitStore.mu.Lock()
	defer circuitStore.mu.Unlock()
	st := getCircuitStateLocked(policy.Policy.Id)
	st.failures++
	threshold := policy.Config.CircuitBreaker.FailureThreshold
	if threshold <= 0 {
		threshold = 5
	}
	if st.failures >= threshold {
		cooldown := policy.Config.CircuitBreaker.CooldownSeconds
		if cooldown <= 0 {
			cooldown = 30
		}
		st.state = CircuitOpen
		st.openedUntil = time.Now().Add(time.Duration(cooldown) * time.Second)
	}
	return st.state
}

func getCircuitStateLocked(id int) *circuitState {
	st := circuitStore.states[id]
	if st == nil {
		st = &circuitState{state: CircuitClosed}
		circuitStore.states[id] = st
	}
	return st
}

func CircuitStats() map[string]any {
	circuitStore.mu.Lock()
	defer circuitStore.mu.Unlock()
	counts := map[string]int{CircuitClosed: 0, CircuitOpen: 0, CircuitHalfOpen: 0}
	for _, st := range circuitStore.states {
		counts[st.state]++
	}
	return map[string]any{
		"states": counts,
		"total":  len(circuitStore.states),
	}
}
