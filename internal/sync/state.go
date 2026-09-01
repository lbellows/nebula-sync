package sync

import (
	"sync"
	"time"
)

const stackSize = 5

// State records the outcome of recent sync runs. It is written by the cron
// goroutine and read by the api health handler, so every access is guarded.
type State struct {
	mu    sync.RWMutex
	stack []Outcome
}

func NewState() *State {
	return &State{
		stack: []Outcome{},
	}
}

type Outcome struct {
	Timestamp time.Time
	Success   bool
}

func NewOutcome(success bool) *Outcome {
	return &Outcome{
		Timestamp: time.Now(),
		Success:   success,
	}
}

func (s *State) Add(outcome Outcome) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stack = append([]Outcome{outcome}, s.stack...)

	if len(s.stack) > stackSize {
		s.stack = s.stack[:stackSize]
	}
}

// Outcomes returns a copy of the recorded outcomes, most recent first.
func (s *State) Outcomes() []Outcome {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]Outcome(nil), s.stack...)
}

// Latest returns the most recent outcome, or false when nothing has run yet.
func (s *State) Latest() (Outcome, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.stack) == 0 {
		return Outcome{}, false
	}

	return s.stack[0], true
}

func (s *State) OnSuccess() {
	s.Add(*NewOutcome(true))
}

func (s *State) OnFailure(err error) {
	s.Add(*NewOutcome(false))
}
