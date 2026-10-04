package health

import (
	"errors"
	"testing"
)

func TestManagerThresholdsAndRecovery(t *testing.T) {
	manager := New(nil, map[string]string{"primary": "primary:7070"}, Config{
		FailureThreshold:  2,
		RecoveryThreshold: 3,
	})

	manager.RecordFailure("primary", errors.New("down"))
	if !manager.IsHealthy("primary") {
		t.Fatal("provider became unhealthy before failure threshold")
	}
	manager.RecordFailure("primary", errors.New("down"))
	if manager.IsHealthy("primary") {
		t.Fatal("provider remained healthy after failure threshold")
	}

	manager.RecordSuccess("primary")
	manager.RecordSuccess("primary")
	if manager.IsHealthy("primary") {
		t.Fatal("provider recovered before recovery threshold")
	}
	manager.RecordSuccess("primary")
	if !manager.IsHealthy("primary") {
		t.Fatal("provider did not recover after recovery threshold")
	}
}

func TestCandidatesPreserveOrderAndSkipUnhealthy(t *testing.T) {
	manager := New(nil, map[string]string{
		"primary": "primary:7070",
		"backup":  "backup:7070",
	}, Config{FailureThreshold: 1})
	manager.RecordFailure("primary", errors.New("down"))

	candidates := manager.Candidates([]string{"primary", "backup"})
	if len(candidates) != 1 || candidates[0] != "backup" {
		t.Fatalf("Candidates() = %#v, want [backup]", candidates)
	}
}
