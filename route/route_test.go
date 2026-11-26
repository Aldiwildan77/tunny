package route

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Aldiwildan77/tunny/observability/observabilitytest"
)

func TestLookup(t *testing.T) {
	table := &Table{
		Routes:   map[string]string{"example.com": "japan", "orphan.example": "japan"},
		Policies: map[string][]string{"policy.example": {"japan", "indonesia"}},
		IPs: map[string]string{
			"203.0.113.10": "japan",
			"203.0.113.20": "indonesia",
			"203.0.113.30": "japan",
		},
		Hosts: map[string]string{
			"203.0.113.10": "example.com",
			"203.0.113.20": "policy.example",
		},
	}

	tests := []struct {
		name   string
		host   string
		want   Match
		wantOK bool
	}{
		{name: "hostname", host: "Example.com", want: Match{Route: "example.com", Providers: []string{"japan"}}, wantOK: true},
		{name: "policy", host: "policy.example", want: Match{Route: "policy.example", Providers: []string{"japan", "indonesia"}}, wantOK: true},
		{name: "ip uses route hostname", host: "203.0.113.10", want: Match{Route: "example.com", Providers: []string{"japan"}}, wantOK: true},
		{name: "ip uses route policy", host: "203.0.113.20", want: Match{Route: "policy.example", Providers: []string{"japan", "indonesia"}}, wantOK: true},
		{name: "ip without hostname falls back to ip", host: "203.0.113.30", want: Match{Route: "203.0.113.30", Providers: []string{"japan"}}, wantOK: true},
		{name: "no route", host: "unrouted.example", want: Match{}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := table.Lookup(tt.host)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRouteCountMetric(t *testing.T) {
	observabilitytest.Install()
	NewWithPolicies(nil, map[string][]string{"policy.example": {"japan", "indonesia"}})

	assert.Equal(t, map[string]float64{
		"tunny.route.kind=hostname": 0,
		"tunny.route.kind=policy":   1,
		"tunny.route.kind=ip":       0,
	}, observabilitytest.Points(t, "tunny.route.count"))
}
