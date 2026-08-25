package model

import "errors"

var (
	ErrReactorNotFound = errors.New("reactor not found")
	ErrReactorBusy     = errors.New("reactor is busy")
	ErrFeedNotMetering = errors.New("feed is not metering")
	ErrFeedBusy        = errors.New("feed is busy")
	ErrTripFailed      = errors.New("emergency trip failed")
	ErrRestoreFailed   = errors.New("restore writeback failed")
	ErrNotTripped      = errors.New("reactor is not tripped")
	ErrValveFailed     = errors.New("valve operation failed")
	ErrWritebackFailed = errors.New("state writeback failed")
	ErrCurveNotFound   = errors.New("process curve not found")
	ErrParamNotFound   = errors.New("process parameter not found")
)
