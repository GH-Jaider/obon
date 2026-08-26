package scan

// DiffState is the lantern lifecycle of a row between refreshes.
type DiffState int

// Row states.
const (
	Stable    DiffState = iota
	Arrived             // just lit: brief warm highlight before settling
	Departing           // drifting away: dimmed for one cycle before removal
)

// Tracker follows which port keys appeared or vanished across cycles so
// the UI can light up newcomers and fade out the departed.
type Tracker struct {
	arrivalCycles uint64 // how many cycles a newcomer stays lit
	states        map[string]*trackEntry
	cycle         uint64
}

type trackEntry struct {
	bornAt   uint64
	lastSeen uint64
	state    DiffState
}

// NewTracker returns a tracker; arrivals stay lit for arrivalCycles cycles.
func NewTracker(arrivalCycles uint64) *Tracker {
	if arrivalCycles == 0 {
		arrivalCycles = 2
	}
	return &Tracker{arrivalCycles: arrivalCycles, states: map[string]*trackEntry{}}
}

// Update feeds the current set of keys and returns each key's state,
// including Departing keys that are no longer present (remove after this cycle).
func (t *Tracker) Update(keys []string) map[string]DiffState {
	t.cycle++
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
		e := t.states[k]
		if e == nil {
			st := Arrived
			if t.cycle == 1 {
				st = Stable // first scan: the river starts calm, nothing "arrives"
			}
			t.states[k] = &trackEntry{bornAt: t.cycle, lastSeen: t.cycle, state: st}
		} else if e.state == Departing {
			e.state = Stable // came back before finishing its send-off
			e.lastSeen = t.cycle
		} else {
			e.lastSeen = t.cycle
			if t.cycle-e.bornAt >= t.arrivalCycles {
				e.state = Stable
			}
		}
	}
	out := make(map[string]DiffState, len(t.states))
	for k, e := range t.states {
		if present[k] {
			out[k] = e.state
			continue
		}
		if e.state == Departing && t.cycle-e.lastSeen >= 1 {
			delete(t.states, k) // drifted far enough; remove
			continue
		}
		e.state = Departing
		out[k] = Departing
	}
	return out
}
