package sqsworker

import (
	"errors"
	"sync"
)

type RunningState int

const (
	RunningStateStopped  RunningState = 0
	RunningStateRunning  RunningState = 1
	RunningStateDraining RunningState = 2
)

type runningState struct {
	state RunningState
	mutex sync.Mutex
}

func (rs *runningState) setRunningState(state RunningState) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.state = state
}

func (rs *runningState) RunningState() RunningState {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	return rs.state
}

type runningState2 struct {
	sync.Mutex
	state RunningState
}

func (rs *runningState2) Running() bool {
	return rs.state == RunningStateRunning
}

func (rs *runningState2) Stopped() bool {
	return rs.state == RunningStateStopped
}

func (rs *runningState2) Draining() bool {
	return rs.state == RunningStateDraining
}

func (rs *runningState2) Set(state RunningState) {
	rs.state = state
}

var errAlreadyInState = errors.New("already in this state")

type runningStateMachine struct {
	mutex sync.Mutex
	state RunningState
}

func (sm *runningStateMachine) State() RunningState {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.state
}

func (sm *runningStateMachine) Set(nextState RunningState) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if nextState == sm.state {
		return errAlreadyInState
	}

	switch nextState {
	case RunningStateRunning:
		switch sm.state {
		case RunningStateDraining:
			return errors.New("cannot start runner while it is draining")
		}

	case RunningStateDraining:
		switch sm.state {
		case RunningStateStopped:
			return errors.New("already stopped")
		}

	case RunningStateStopped:
		switch sm.state {
		case RunningStateRunning:
			return errors.New("need to drain first")
		}
	}

	sm.state = nextState
	return nil
}
