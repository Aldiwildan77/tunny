package health

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Aldiwildan77/tunny/node"
)

type State string

const (
	StateHealthy   State = "healthy"
	StateUnhealthy State = "unhealthy"
)

type Config struct {
	Enabled           bool
	Interval          time.Duration
	Timeout           time.Duration
	FailureThreshold  int
	RecoveryThreshold int
}

type Status struct {
	Provider             string
	Address              string
	State                State
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	LastCheck            time.Time
	LastTransition       time.Time
	LastError            string
}

type Manager struct {
	node      node.Node
	providers map[string]string
	config    Config

	mu       sync.RWMutex
	statuses map[string]*Status
}

func New(n node.Node, providers map[string]string, cfg Config) *Manager {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 2
	}
	if cfg.RecoveryThreshold <= 0 {
		cfg.RecoveryThreshold = 3
	}

	addresses := make(map[string]string, len(providers))
	statuses := make(map[string]*Status, len(providers))
	for name, address := range providers {
		addresses[name] = address
		statuses[name] = &Status{Provider: name, Address: address, State: StateHealthy}
	}

	return &Manager{node: n, providers: addresses, config: cfg, statuses: statuses}
}

func (m *Manager) Start(ctx context.Context) {
	if !m.config.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(m.config.Interval)
		defer ticker.Stop()
		m.checkAll(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkAll(ctx)
			}
		}
	}()
}

func (m *Manager) checkAll(ctx context.Context) {
	for name := range m.providers {
		checkCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
		err := m.check(checkCtx, name)
		cancel()
		if err != nil {
			m.RecordFailure(name, err)
		} else {
			m.RecordSuccess(name)
		}
	}
}

func (m *Manager) check(ctx context.Context, name string) error {
	address, ok := m.providers[name]
	if !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	conn, err := m.node.Dial(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("PING\n")); err != nil {
		return err
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(response) != "PONG" {
		return fmt.Errorf("unexpected health response: %q", strings.TrimSpace(response))
	}
	return nil
}

func (m *Manager) RecordFailure(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, ok := m.statuses[name]
	if !ok {
		return
	}
	status.LastCheck = time.Now()
	status.ConsecutiveFailures++
	status.ConsecutiveSuccesses = 0
	status.LastError = err.Error()
	if status.State != StateUnhealthy && status.ConsecutiveFailures >= m.config.FailureThreshold {
		status.State = StateUnhealthy
		status.LastTransition = status.LastCheck
	}
}

func (m *Manager) RecordSuccess(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, ok := m.statuses[name]
	if !ok {
		return
	}
	status.LastCheck = time.Now()
	status.ConsecutiveFailures = 0
	status.ConsecutiveSuccesses++
	status.LastError = ""
	if status.State == StateUnhealthy && status.ConsecutiveSuccesses >= m.config.RecoveryThreshold {
		status.State = StateHealthy
		status.LastTransition = status.LastCheck
	}
}

func (m *Manager) IsHealthy(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.statuses[name]
	return ok && status.State != StateUnhealthy
}

func (m *Manager) Candidates(names []string) []string {
	candidates := make([]string, 0, len(names))
	for _, name := range names {
		if m.IsHealthy(name) {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

func (m *Manager) Snapshot() []Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Status, 0, len(m.statuses))
	for _, status := range m.statuses {
		result = append(result, *status)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Provider < result[j].Provider })
	return result
}
