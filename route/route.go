package route

import (
	"net"
	"strings"
	"sync"
)

type Table struct {
	Routes map[string]string
	IPs    map[string]string
	Hosts  map[string]string

	mu sync.RWMutex
}

func New(routes map[string]string) *Table {
	table := &Table{
		Routes: routes,
		IPs:    make(map[string]string),
		Hosts:  make(map[string]string),
	}

	for hostname, provider := range routes {
		hostname = strings.ToLower(hostname)

		table.Routes[hostname] = provider

		ips, err := net.LookupHost(hostname)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			table.IPs[ip] = provider
			table.Hosts[ip] = hostname
		}
	}

	return table
}

func (t *Table) GetRoute(hostname string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hostname = strings.ToLower(hostname)

	provider, ok := t.Routes[hostname]
	return provider, ok
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

func (t *Table) GetIPs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ips := make([]string, 0, len(t.IPs))

	for ip := range t.IPs {
		ips = append(ips, ip)
	}

	return ips
}

func (t *Table) GetNetIPs() []net.IP {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ips := make([]net.IP, 0, len(t.IPs))

	for ipStr := range t.IPs {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

func (t *Table) GetHost(ip string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	host, ok := t.Hosts[ip]
	return host, ok
}
