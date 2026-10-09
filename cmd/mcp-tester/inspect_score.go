package main

// listFailurePenalty is deducted for each list request that fails although its
// capability is declared: clients break on it, so it outweighs any style hint.
const listFailurePenalty = 50

// safetyBonusCap caps the bonus for readOnlyHint annotations (2 per tool).
const safetyBonusCap = 20

// deductions sums penalties per category and caps each category, so that a
// server with many tools is not punished for the same mistake without bound.
type deductions struct {
	sums map[string]int
	caps map[string]int
}

// specCategories are MUST violations: they are deducted after the score is
// capped at 100, so the safety bonus cannot hide them.
var specCategories = map[string]bool{
	"listFailure": true,
	"schemaType":  true,
	"cache":       true,
	"xMCPHeader":  true,
	"skills":      true,
	"tasks":       true,
}

// inspectDeductionCaps are the per-category caps of inspect.
func inspectDeductionCaps() map[string]int {
	return map[string]int{
		"description":  20,
		"outputSchema": 10,
		"protocol":     30,
		"toolName":     15,
		"duplicate":    10,
		"schemaType":   15,
		"title":        5,
		"icon":         15,
		"cache":        3,
		"cacheScope":   10,
		"xMCPHeader":   20,
	}
}

func newDeductions(caps map[string]int) *deductions {
	return &deductions{sums: map[string]int{}, caps: caps}
}

func (d *deductions) add(category string, points int) { d.sums[category] += points }

// total sums the capped categories, either the spec violations or the rest.
func (d *deductions) total(spec bool) int {
	total := 0
	for category, sum := range d.sums {
		if specCategories[category] != spec {
			continue
		}
		if limit, ok := d.caps[category]; ok && sum > limit {
			sum = limit
		}
		total += sum
	}
	return total
}

// finalScore applies the quality deductions to score (which carries the
// safety bonus), caps it at 100, then deducts the spec violations.
func finalScore(score int, d *deductions) int {
	score = min(score-d.total(false), 100)
	return max(score-d.total(true), 0)
}
