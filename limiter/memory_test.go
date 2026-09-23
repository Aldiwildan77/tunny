package limiter_test

import (
	"testing"

	"github.com/Aldiwildan77/tunny/limiter"
)

func TestMemoryLimiter_Set(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		limit   int
		wantErr bool
	}{
		{
			name:    "GIVEN a valid key and limit WHEN Set is called THEN it should not return an error",
			key:     "valid_key",
			limit:   2,
			wantErr: false,
		},
		{
			name:    "GIVEN an empty key WHEN Set is called THEN it should return an error",
			key:     "",
			limit:   2,
			wantErr: true,
		},
		{
			name:    "GIVEN a zero limit WHEN Set is called THEN it should return an error",
			key:     "valid_key",
			limit:   0,
			wantErr: true,
		},
		{
			name:    "GIVEN a negative limit WHEN Set is called THEN it should return an error",
			key:     "valid_key",
			limit:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memoryLimiter limiter.MemoryLimiter

			err := memoryLimiter.Set(tt.key, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryLimiter_Acquire(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		acquires int
		want     []bool
	}{
		{
			name:     "GIVEN a limit of one WHEN acquiring once THEN it should succeed",
			limit:    1,
			acquires: 1,
			want:     []bool{true},
		},
		{
			name:     "GIVEN a limit of one WHEN acquiring twice THEN the second acquire should fail",
			limit:    1,
			acquires: 2,
			want:     []bool{true, false},
		},
		{
			name:     "GIVEN a missing key WHEN Acquire is called THEN it should fail",
			limit:    0,
			acquires: 1,
			want:     []bool{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var memoryLimiter limiter.MemoryLimiter

			if tt.limit > 0 {
				if err := memoryLimiter.Set("test_key", tt.limit); err != nil {
					t.Fatal(err)
				}
			}

			for index, want := range tt.want {
				got := memoryLimiter.Acquire("test_key")
				if got != want {
					t.Fatalf("Acquire() call %d = %v, want %v", index+1, got, want)
				}
			}
		})
	}
}

func TestMemoryLimiter_Release(t *testing.T) {
	t.Run("GIVEN an acquired slot WHEN Release is called THEN the slot should become available", func(t *testing.T) {
		var memoryLimiter limiter.MemoryLimiter

		if err := memoryLimiter.Set("test_key", 1); err != nil {
			t.Fatal(err)
		}

		if !memoryLimiter.Acquire("test_key") {
			t.Fatal("first Acquire() should succeed")
		}

		memoryLimiter.Release("test_key")

		if !memoryLimiter.Acquire("test_key") {
			t.Fatal("Acquire() should succeed after Release()")
		}
	})

	t.Run("GIVEN a missing key WHEN Release is called THEN it should not panic", func(t *testing.T) {
		var memoryLimiter limiter.MemoryLimiter

		memoryLimiter.Release("missing_key")
	})
}

func TestMemoryLimiter_Remove(t *testing.T) {
	t.Run("GIVEN an existing key WHEN Remove is called THEN it should remove the limiter", func(t *testing.T) {
		var memoryLimiter limiter.MemoryLimiter

		if err := memoryLimiter.Set("test_key", 1); err != nil {
			t.Fatal(err)
		}

		if err := memoryLimiter.Remove("test_key"); err != nil {
			t.Fatal(err)
		}

		if memoryLimiter.Acquire("test_key") {
			t.Fatal("Acquire() should fail after Remove()")
		}
	})

	t.Run("GIVEN a missing key WHEN Remove is called THEN it should not return an error", func(t *testing.T) {
		var memoryLimiter limiter.MemoryLimiter

		if err := memoryLimiter.Remove("missing_key"); err != nil {
			t.Fatalf("Remove() returned unexpected error: %v", err)
		}
	})
}
