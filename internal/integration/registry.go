package integration

import (
	"fmt"
	"sort"
	"sync"
)

// ConnectorFactory creates a fresh connector instance for registry lookups.
type ConnectorFactory func() Connector

// Registry stores connector factories by stable provider name.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ConnectorFactory
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]ConnectorFactory)}
}

// Register adds a connector factory under name.
func (r *Registry) Register(name string, factory ConnectorFactory) error {
	if r == nil {
		return fmt.Errorf("integration registry is nil")
	}
	if name == "" {
		return fmt.Errorf("connector name is required")
	}
	if factory == nil {
		return fmt.Errorf("connector factory is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("connector %q already registered", name)
	}
	r.factories[name] = factory
	return nil
}

// Get returns a new connector instance for name.
func (r *Registry) Get(name string) (Connector, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return factory(), true
}

// Names returns registered connector names in insertion-independent map order.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
