package route

import (
	"maps"
	"net"
	"strings"
	"sync"
)

type Table struct {
	Routes   map[string]string
	Policies map[string][]string
	IPs      map[string]string
	Hosts    map[string]string

	mu sync.RWMutex
}

func New(routes map[string]string) *Table {
	return NewWithPolicies(routes, nil)
}

func NewWithPolicies(routes map[string]string, policies map[string][]string) *Table {
	if routes == nil {
		routes = make(map[string]string)
	}
	if policies == nil {
		policies = make(map[string][]string)
	}
	normalizedPolicies := make(map[string][]string, len(policies))
	for hostname, providers := range policies {
		normalizedPolicies[strings.ToLower(hostname)] = append([]string(nil), providers...)
	}

	table := &Table{
		Routes:   routes,
		Policies: normalizedPolicies,
		IPs:      make(map[string]string),
		Hosts:    make(map[string]string),
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

func (t *Table) MatchCandidates(host string) ([]string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	host = strings.ToLower(host)
	if providers, ok := t.Policies[host]; ok && len(providers) > 0 {
		return append([]string(nil), providers...), true
	}
	if provider, ok := t.Routes[host]; ok {
		return []string{provider}, true
	}
	if provider, ok := t.IPs[host]; ok {
		if routeHost, exists := t.Hosts[host]; exists {
			if providers, policyExists := t.Policies[routeHost]; policyExists && len(providers) > 0 {
				return append([]string(nil), providers...), true
			}
		}
		return []string{provider}, true
	}
	return nil, false
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
	providers, ok := t.MatchCandidates(host)
	if ok && len(providers) > 0 {
		return providers[0], true
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

func (t *Table) Snapshot() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	routes := make(map[string]string, len(t.Routes))
	maps.Copy(routes, t.Routes)

	return routes
}
