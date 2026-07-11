package core

import (
	"context"
	"sync"
)

type MockManager struct {
	mu     sync.Mutex
	status Status
}

func NewMockManager(initial Status) *MockManager {
	if initial == "" {
		initial = StatusStopped
	}
	return &MockManager{status: initial}
}

func (m *MockManager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *MockManager) Start(ctx context.Context) error {
	m.mu.Lock()
	m.status = StatusStarting
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		m.mu.Lock()
		m.status = StatusFailed
		m.mu.Unlock()
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	m.status = StatusRunning
	m.mu.Unlock()
	return nil
}

func (m *MockManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusStopped
	return nil
}

func (m *MockManager) Restart(ctx context.Context) error {
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start(ctx)
}
