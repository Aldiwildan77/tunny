package route

import (
	"net"
	"strings"
	"sync"
)

type Table struct {
	Routes map[string]string
	mu     sync.RWMutex
}

func New(routes map[string]string) *Table {
	return &Table{
		Routes: routes,
	}
}

func (t *Table) GetRoute(hostname string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	route, ok := t.Routes[hostname]
	return route, ok
}

func (t *Table) AddRoute(hostname, route string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Routes[hostname] = route
}

func (t *Table) RemoveRoute(hostname string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.Routes, hostname)
}

func (t *Table) Match(route string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	route = strings.ToLower(route)

	if provider, ok := t.Routes[route]; ok {
		return provider, true
	}

	// IP address matching
	if net.ParseIP(route) != nil {
		if provider, ok := t.Routes[route]; ok {
			return provider, true
		}
	}

	return "", false
}
