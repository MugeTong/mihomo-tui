package core

import "context"

type Status string

const (
	StatusUnavailable Status = "unavailable"
	StatusStopped     Status = "stopped"
	StatusStarting    Status = "starting"
	StatusRunning     Status = "running"
	StatusStopping    Status = "stopping"
	StatusFailed      Status = "failed"
)

type Manager interface {
	Status() Status
	Start(ctx context.Context) error
	Stop() error
}
