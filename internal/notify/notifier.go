package notify

import "time"

type Payload struct {
	Name      string
	URL       string
	Status    string
	LatencyMs int64
	Failures  int
	Reason    string
	Timestamp time.Time
}

type Notifier interface {
	Name() string
	SendCritical(p Payload) error
	SendRecovery(p Payload) error
}
