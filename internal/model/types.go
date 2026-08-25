package model

type ReactorState string

const (
	ReactorIdle     ReactorState = "idle"
	ReactorHeating  ReactorState = "heating"
	ReactorReacting ReactorState = "reacting"
	ReactorCooling  ReactorState = "cooling"
)

type InterlockState string

const (
	InterlockFree        InterlockState = "free"
	InterlockInterlocked InterlockState = "interlocked"
	InterlockTripped     InterlockState = "tripped"
	InterlockReset       InterlockState = "reset"
)

type FeedState string

const (
	FeedIdle     FeedState = "idle"
	FeedMetering FeedState = "metering"
	FeedFeeding  FeedState = "feeding"
	FeedStopped  FeedState = "stopped"
)

type Reactor struct {
	ID          string
	State       ReactorState
	Temperature float64
	Pressure    float64
	Tripped     bool
	Restored    bool
	Manual      bool
}

type ProcessParams struct {
	TargetTemperature float64
	TargetPressure    float64
	FeedTarget        float64
	CoolTarget        float64
	CurveID           string
}

type CurveStep struct {
	Index       int
	Temperature float64
	Pressure    float64
	HoldSeconds int
}

type Curve struct {
	ID    string
	Steps []CurveStep
}

type FeedConfirmation struct {
	Amount float64
	OK     bool
}

type Alarm struct {
	ReactorID string
	Level     string
	Message   string
	Active    bool
}

type RecordEntry struct {
	ReactorID   string
	State       ReactorState
	Temperature float64
	Pressure    float64
	Seq         uint64
}
