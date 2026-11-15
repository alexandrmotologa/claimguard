package verifier

import "math"

// IsWithinTolerance checks if the asserted value is within acceptable delta of the expected value.
// It checks both the fact-specific tolerance (absolute) and the global percentage tolerance.
func IsWithinTolerance(asserted, expected, factTolerance, globalTolerancePct float64) bool {
	diff := math.Abs(asserted - expected)

	// If absolute difference is zero or within fact-specific tolerance
	if diff == 0 || diff <= factTolerance {
		return true
	}

	// Check relative percentage tolerance if configured
	if globalTolerancePct > 0 && expected != 0 {
		relDiff := diff / math.Abs(expected)
		if relDiff <= globalTolerancePct {
			return true
		}
	}

	return false
}
