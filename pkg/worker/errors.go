package worker

import "errors"

var ErrRuntimeDraining = errors.New("runtime is currently draining")
var ErrRuntimeIdle = errors.New("runtime is currently idle")
var ErrRuntimeActive = errors.New("runtime is currently active")
