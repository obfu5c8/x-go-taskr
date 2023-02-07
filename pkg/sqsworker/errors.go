package sqsworker

import "errors"

var ErrNoNewMessages = errors.New("no new messages")
var ErrShutdownRequested = errors.New("shutdown requested")

var ErrMessageAlreadyProcessed = errors.New("message already processed")
var ErrMessageLocked = errors.New("message locked elsewhere")
