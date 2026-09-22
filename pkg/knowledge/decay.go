package knowledge

import (
	"math"
	"time"
)

// CalculateDecayedWeight applies the exponential decay formula based on age
// Weight = OriginalWeight * 2^(-ageDays / HalfLifeDays); preserve rejection penalties.
func CalculateDecayedWeight(decay KnowledgeDecay, now time.Time, originalWeight float64) float64 {
	// If the half life is zero or negative, do not decay (infinite)
	if decay.HalfLifeDays <= 0 {
		return originalWeight
	}

	// Age is calculated from the LastConfirmed timestamp
	ageDuration := now.Sub(decay.LastConfirmed)
	ageDays := ageDuration.Hours() / 24.0

	// If it's negative age (time traveling, tests, or just now), age is 0
	if ageDays < 0 {
		ageDays = 0
	}

	// Calculate decay multiplier
	multiplier := math.Exp(-math.Ln2 * ageDays / float64(decay.HalfLifeDays))
	
	newWeight := originalWeight * multiplier

	// Only positive evidence uses a positive minimum. Never turn rejected
	// evidence (a negative weight) or a zero-weight observation into a boost.
	if originalWeight > 0 && decay.MinWeight > 0 && newWeight < decay.MinWeight {
		return decay.MinWeight
	}

	return newWeight
}
