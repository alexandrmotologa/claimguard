package extractor

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ValueSpan represents an extracted numerical value and its location within a sentence.
type ValueSpan struct {
	RawMatch     string
	NumericValue float64
	Unit         string
	ClaimType    string // currency, duration, percentage, number
	Start        int
	End          int
}

// Regex patterns for entity and numerical claim extraction.
var (
	// Currency patterns: $1,500, $499, $499/month, $29/mo, 499 USD, 50 euros, 50 dollars
	reCurrencyPrefix = regexp.MustCompile(`(?i)([\$€£])\s*([0-9]+(?:,[0-9]{3})*(?:\.[0-9]+)?)(?:\s*(?:/|\s+per\s+)?(month|mo|year|yr|day|order))?`)
	reCurrencySuffix = regexp.MustCompile(`(?i)\b([0-9]+(?:,[0-9]{3})*(?:\.[0-9]+)?)\s*(usd|eur|gbp|dollars|euros|cents)(?:\s*(?:/|\s+per\s+)?(month|mo|year|yr|day|order))?\b`)

	// Duration patterns: 14 days, 60-day, 24 hours, 3 months, 1 year
	reDuration = regexp.MustCompile(`(?i)\b([0-9]+(?:,[0-9]{3})*(?:\.[0-9]+)?)(?:-|\s+)(days?|hours?|months?|years?|weeks?)\b`)

	// Percentage patterns: 18.5%, 99.95%, 20 percent, 20%
	rePercentage = regexp.MustCompile(`(?i)\b([0-9]+(?:\.[0-9]+)?)\s*(%|percent(?:age points?)?)`)
)

func cleanNumber(raw string) string {
	return strings.ReplaceAll(raw, ",", "")
}

// ExtractNumericalSpans scans a sentence for verifiable numbers (currencies, durations, percentages).
func ExtractNumericalSpans(sentence string) []ValueSpan {
	var spans []ValueSpan
	occupied := make([]bool, len(sentence))

	isOccupied := func(start, end int) bool {
		for i := start; i < end && i < len(occupied); i++ {
			if occupied[i] {
				return true
			}
		}
		return false
	}

	markOccupied := func(start, end int) {
		for i := start; i < end && i < len(occupied); i++ {
			occupied[i] = true
		}
	}

	// 1. Currency prefix ($499/month, €50, etc.)
	for _, m := range reCurrencyPrefix.FindAllStringSubmatchIndex(sentence, -1) {
		start, end := m[0], m[1]
		if isOccupied(start, end) {
			continue
		}
		sym := sentence[m[2]:m[3]]
		numStr := cleanNumber(sentence[m[4]:m[5]])
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue
		}

		unit := "USD"
		if sym == "€" {
			unit = "EUR"
		} else if sym == "£" {
			unit = "GBP"
		}

		if m[6] != -1 && m[7] != -1 {
			period := strings.ToLower(sentence[m[6]:m[7]])
			if period == "mo" || period == "month" {
				unit += "/month"
			} else if period == "yr" || period == "year" {
				unit += "/year"
			}
		}

		spans = append(spans, ValueSpan{
			RawMatch:     sentence[start:end],
			NumericValue: val,
			Unit:         unit,
			ClaimType:    "currency",
			Start:        start,
			End:          end,
		})
		markOccupied(start, end)
	}

	// 2. Currency suffix (499 USD, 20 dollars, etc.)
	for _, m := range reCurrencySuffix.FindAllStringSubmatchIndex(sentence, -1) {
		start, end := m[0], m[1]
		if isOccupied(start, end) {
			continue
		}
		numStr := cleanNumber(sentence[m[2]:m[3]])
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue
		}

		currWord := strings.ToLower(sentence[m[4]:m[5]])
		unit := "USD"
		if currWord == "eur" || currWord == "euros" {
			unit = "EUR"
		} else if currWord == "gbp" {
			unit = "GBP"
		}

		if m[6] != -1 && m[7] != -1 {
			period := strings.ToLower(sentence[m[6]:m[7]])
			if period == "mo" || period == "month" {
				unit += "/month"
			}
		}

		spans = append(spans, ValueSpan{
			RawMatch:     sentence[start:end],
			NumericValue: val,
			Unit:         unit,
			ClaimType:    "currency",
			Start:        start,
			End:          end,
		})
		markOccupied(start, end)
	}

	// 3. Durations (14 days, 60-day, etc.)
	for _, m := range reDuration.FindAllStringSubmatchIndex(sentence, -1) {
		start, end := m[0], m[1]
		if isOccupied(start, end) {
			continue
		}
		numStr := cleanNumber(sentence[m[2]:m[3]])
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue
		}

		unit := strings.ToLower(sentence[m[4]:m[5]])
		spans = append(spans, ValueSpan{
			RawMatch:     sentence[start:end],
			NumericValue: val,
			Unit:         unit,
			ClaimType:    "duration",
			Start:        start,
			End:          end,
		})
		markOccupied(start, end)
	}

	// 4. Percentage (18.5%, 99.95%)
	for _, m := range rePercentage.FindAllStringSubmatchIndex(sentence, -1) {
		start, end := m[0], m[1]
		if isOccupied(start, end) {
			continue
		}
		numStr := cleanNumber(sentence[m[2]:m[3]])
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			continue
		}

		spans = append(spans, ValueSpan{
			RawMatch:     sentence[start:end],
			NumericValue: val,
			Unit:         "%",
			ClaimType:    "percentage",
			Start:        start,
			End:          end,
		})
		markOccupied(start, end)
	}

	// Sort spans in ascending order of start position
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].Start < spans[j].Start
	})

	return spans
}
