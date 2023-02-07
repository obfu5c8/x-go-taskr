package worker

import (
	"fmt"
	"sync"
)

type State int

const (
	StateIdle     State = 0
	StateActive   State = 1
	StateDraining State = 2
)

type StateMachine interface {
	Get() State
	Is(State) bool
	TrySet(State) error
}

type runtimeState struct {
	mu    *sync.RWMutex
	state State
}

func (s *runtimeState) Get() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.state
}

func (s *runtimeState) Is(state State) bool {
	return s.Get() == state
}

func (s *runtimeState) TrySet(state State) error {
	switch state {
	case StateActive:
		return s.trySetActive()
	case StateIdle:
		return s.trySetIdle()
	case StateDraining:
		return s.trySetDraining()
	}

	return fmt.Errorf("unrecognised desired state: %v", state)
}

func (s *runtimeState) trySetActive() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateDraining {
		return ErrRuntimeDraining
	}

	s.state = StateActive
	return nil
}

func (s *runtimeState) trySetDraining() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case StateDraining:
		return ErrRuntimeDraining
	case StateIdle:
		return ErrRuntimeIdle
	}

	s.state = StateDraining
	return nil
}

func (s *runtimeState) trySetIdle() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateActive {
		return ErrRuntimeActive
	}
	return nil
}
