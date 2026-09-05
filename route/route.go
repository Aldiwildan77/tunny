package route

import (
	"net"
	"strings"
	"sync"
)

type Table struct {
	Routes map[string]string
	IPs    map[string]string

	mu sync.RWMutex
}

func New(routes map[string]string) *Table {
	table := &Table{
		Routes: routes,
		IPs:    make(map[string]string),
	}

	for hostname, provider := range routes {
		hostname = strings.ToLower(hostname)

		ips, err := net.LookupHost(hostname)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			table.IPs[ip] = provider
		}
	}

	return table
}

func (t *Table) GetRoute(hostname string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hostname = strings.ToLower(hostname)

	route, ok := t.Routes[hostname]
	return route, ok
}

func (t *Table) AddRoute(hostname, provider string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	hostname = strings.ToLower(hostname)

	t.Routes[hostname] = provider

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return
	}

	for _, ip := range ips {
		t.IPs[ip] = provider
	}
}

func (t *Table) RemoveRoute(hostname string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	hostname = strings.ToLower(hostname)

	delete(t.Routes, hostname)

	// Remove IP mappings belonging to this provider/hostname.
	// This is intentionally kept simple for now.
}

func (t *Table) Match(host string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	host = strings.ToLower(host)

	// Hostname match
	if provider, ok := t.Routes[host]; ok {
		return provider, true
	}

	// IP match
	if provider, ok := t.IPs[host]; ok {
		return provider, true
	}

	return "", false
}
