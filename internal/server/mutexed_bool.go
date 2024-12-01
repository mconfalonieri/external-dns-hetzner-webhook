package server

import "sync"

// mutexedBool is a mutex-protected boolean value.
type mutexedBool struct {
	m sync.Mutex
	v bool
}

// Set sets the value for this instance.
func (b *mutexedBool) Set(v bool) {
	b.m.Lock()
	b.v = v
	b.m.Unlock()
}

// Get gets the value from this instance.
func (b *mutexedBool) Get() bool {
	var v bool
	b.m.Lock()
	v = b.v
	b.m.Unlock()
	return v
}
