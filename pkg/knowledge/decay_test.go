package knowledge

import (
	"testing"
	"time"
)

func TestKnowledgeDecayHalfLife(t *testing.T) {
	decay := KnowledgeDecay{
		LastConfirmed: time.Now().Add(-180 * 24 * time.Hour), // 180 days ago
		LastObserved:  time.Now().Add(-10 * 24 * time.Hour),  // 10 days ago
		HalfLifeDays:  90,
		MinWeight:     10.0,
	}

	// 180 days age = 2 half lives. 2^(-2) = 0.25
	originalWeight := 100.0
	current := CalculateDecayedWeight(decay, time.Now(), originalWeight)

	if current > 25.1 || current < 24.9 {
		t.Fatalf("Decay weight math incorrect, got %f expected ~25", current)
	}
}

func TestKnowledgeDecayMinWeight(t *testing.T) {
	decay := KnowledgeDecay{
		LastConfirmed: time.Now().Add(-1000 * 24 * time.Hour),
		HalfLifeDays:  90,
		MinWeight:     5.0,
	}

	current := CalculateDecayedWeight(decay, time.Now(), 100.0)
	if current != 5.0 {
		t.Fatalf("Decay failed to respect min weight, got %f expected 5.0", current)
	}
}

func TestKnowledgeDecayNeverTurnsRejectionIntoBoost(t *testing.T) {
	at := time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)
	decay := KnowledgeDecay{LastConfirmed: at.Add(-30 * 24 * time.Hour), HalfLifeDays: 30, MinWeight: 5}

	if got := CalculateDecayedWeight(decay, at, -10); got < -5.01 || got > -4.99 {
		t.Fatalf("negative weight after one half-life = %f, want -5", got)
	}
	if got := CalculateDecayedWeight(decay, at, 0); got != 0 {
		t.Fatalf("zero evidence became positive: %f", got)
	}
}
