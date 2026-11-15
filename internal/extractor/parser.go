package extractor

import (
	"strings"
	"unicode"
)

// SentenceSpan represents a single segmented sentence within a larger text stream.
type SentenceSpan struct {
	Text      string
	StartByte int
	EndByte   int
	HasTerm   bool // True if sentence ends with terminal punctuation (. ? ! \n)
}

var commonAbbreviations = map[string]bool{
	"mr.":   true,
	"mrs.":  true,
	"ms.":   true,
	"dr.":   true,
	"prof.": true,
	"e.g.":  true,
	"i.e.":  true,
	"u.s.":  true,
	"vs.":   true,
	"etc.":  true,
	"inc.":  true,
	"ltd.":  true,
	"corp.": true,
}

// isDecimalNumber checks if the period at index i is part of a float (e.g., "14.5" or "$49.99").
func isDecimalNumber(s string, i int) bool {
	if i <= 0 || i >= len(s)-1 {
		return false
	}
	prev := s[i-1]
	next := s[i+1]
	return unicode.IsDigit(rune(prev)) && unicode.IsDigit(rune(next))
}

// isAbbreviation checks if the period at index i is part of a recognized abbreviation.
func isAbbreviation(s string, i int) bool {
	// Look backwards up to 8 bytes for an abbreviation word
	start := i
	for start > 0 && (unicode.IsLetter(rune(s[start-1])) || s[start-1] == '.') {
		start--
	}
	word := strings.ToLower(s[start : i+1])
	return commonAbbreviations[word]
}

// SplitSentences parses text into discrete sentences while respecting abbreviations, decimals, and punctuation.
// If isEOF is false, any trailing incomplete sentence without terminal punctuation is omitted (held in sliding window).
func SplitSentences(text string, isEOF bool) ([]SentenceSpan, string) {
	var sentences []SentenceSpan
	n := len(text)
	if n == 0 {
		return sentences, ""
	}

	start := 0
	i := 0

	for i < n {
		b := text[i]

		// Check for newline boundaries
		if b == '\n' {
			// Check if double newline or sentence ending
			spanText := strings.TrimSpace(text[start : i+1])
			if spanText != "" {
				sentences = append(sentences, SentenceSpan{
					Text:      spanText,
					StartByte: start,
					EndByte:   i + 1,
					HasTerm:   true,
				})
			}
			// Skip whitespace
			i++
			for i < n && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r' || text[i] == '\n') {
				i++
			}
			start = i
			continue
		}

		// Check for punctuation terminals: . ? !
		if b == '.' || b == '?' || b == '!' {
			// If period, check if it's decimal point or abbreviation
			if b == '.' {
				if isDecimalNumber(text, i) || isAbbreviation(text, i) {
					i++
					continue
				}
			}

			// Check if followed by whitespace, quote, or end of string
			isBoundary := false
			nextIdx := i + 1
			if nextIdx >= n {
				if isEOF {
					isBoundary = true
				}
			} else {
				nextChar := text[nextIdx]
				if unicode.IsSpace(rune(nextChar)) || nextChar == '"' || nextChar == '\'' || nextChar == ')' {
					isBoundary = true
				}
			}

			if isBoundary {
				// Consume terminal punctuation and any trailing quotes/closing parens
				end := nextIdx
				for end < n && (text[end] == '"' || text[end] == '\'' || text[end] == ')') {
					end++
				}

				spanText := strings.TrimSpace(text[start:end])
				if spanText != "" {
					sentences = append(sentences, SentenceSpan{
						Text:      spanText,
						StartByte: start,
						EndByte:   end,
						HasTerm:   true,
					})
				}

				// Move start past trailing spaces
				i = end
				for i < n && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r') {
					i++
				}
				start = i
				continue
			}
		}

		i++
	}

	// Handle leftover incomplete buffer
	var remainder string
	if start < n {
		trailing := text[start:]
		if isEOF {
			trimmed := strings.TrimSpace(trailing)
			if trimmed != "" {
				sentences = append(sentences, SentenceSpan{
					Text:      trimmed,
					StartByte: start,
					EndByte:   n,
					HasTerm:   false,
				})
			}
			remainder = ""
		} else {
			remainder = trailing
		}
	}

	return sentences, remainder
}
