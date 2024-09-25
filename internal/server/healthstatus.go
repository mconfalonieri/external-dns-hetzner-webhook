package server

import "sync"

// HealthStatus contains the health and ready statuses for the webhook.
type HealthStatus struct {
	m       sync.Mutex
	healthy bool
	ready   bool
}

// SetHealth sets the health status.
func (h *HealthStatus) SetHealth(v bool) {
	h.m.Lock()
	h.healthy = v
	h.m.Unlock()
}

// SetReady sets the readiness status.
func (h *HealthStatus) SetReady(v bool) {
	h.m.Lock()
	h.ready = v
	h.m.Unlock()
}

// IsHealthy returns the healthy flag.
func (h *HealthStatus) IsHealthy() bool {
	var healthy bool
	h.m.Lock()
	healthy = h.healthy
	h.m.Unlock()
	return healthy
}

// IsReady returns the readiness status.
func (h *HealthStatus) IsReady() bool {
	var ready bool
	h.m.Lock()
	ready = h.ready
	h.m.Unlock()
	return ready
}
