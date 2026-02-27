package domain

import (
	"testing"
	"time"
)

func TestComputeStatus(t *testing.T) {
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	nextDue := now

	if got := ComputeStatus(now, nextDue, 180, false); got != StatusUp {
		t.Fatalf("expected up at due boundary, got %s", got)
	}
	if got := ComputeStatus(now.Add(1*time.Minute), nextDue, 180, false); got != StatusLate {
		t.Fatalf("expected late in grace window, got %s", got)
	}
	if got := ComputeStatus(now.Add(4*time.Minute), nextDue, 180, false); got != StatusDown {
		t.Fatalf("expected down after grace, got %s", got)
	}
	if got := ComputeStatus(now.Add(10*time.Minute), nextDue, 180, true); got != StatusPaused {
		t.Fatalf("expected paused to dominate, got %s", got)
	}
}
