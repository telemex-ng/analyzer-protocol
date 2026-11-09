// Package values contains the small, shared parsing helpers used by both the
// HL7 and ASTM analyzers. Keeping them here avoids two subtly different
// interpretations of a numeric result, reference range or timestamp.
package values

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	numericPrefix = regexp.MustCompile(`^[<>]=?\s*`)
	rangePattern  = regexp.MustCompile(`^\s*([<>]?)\s*(-?\d+(?:\.\d+)?)\s*(?:-|–|—|to)\s*([<>]?)\s*(-?\d+(?:\.\d+)?)\s*$`)
	singlePattern = regexp.MustCompile(`^\s*([<>])\s*(-?\d+(?:\.\d+)?)\s*$`)
	timestampHead = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})(\d{2})?(\d{2})?(\d{2})?`)
)

// Numeric returns a float for a plain numeric observation. Values such as
// "<0.5" or ">100" still yield the boundary number; the abnormal flag carried
// by the message remains authoritative.
func Numeric(raw string) *float64 {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, ",", "")
	text = numericPrefix.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil
	}
	return &n
}

// Range parses the common reference-range shapes used by analyzers:
// "4.0-10.0", "4.0 - 10.0", "4.0 to 10.0", "<5" and ">2". A single bound is
// returned as the low or high side respectively; the other side is nil.
func Range(raw string) (low, high *float64) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, nil
	}
	text = strings.ReplaceAll(text, ",", "")
	if match := rangePattern.FindStringSubmatch(text); match != nil {
		first, err1 := strconv.ParseFloat(match[2], 64)
		second, err2 := strconv.ParseFloat(match[4], 64)
		if err1 == nil && err2 == nil {
			switch {
			case match[1] == "<" || match[3] == "<":
				return nil, &first
			case match[1] == ">" || match[3] == ">":
				return &first, nil
			default:
				return &first, &second
			}
		}
	}
	if match := singlePattern.FindStringSubmatch(text); match != nil {
		n, err := strconv.ParseFloat(match[2], 64)
		if err == nil {
			if match[1] == "<" {
				return nil, &n
			}
			return &n, nil
		}
	}
	return nil, nil
}

// Timestamp parses HL7/ASTM compact timestamps (YYYYMMDDHHMMSS, optionally
// with a fractional second and timezone) and RFC3339. Analyzer timestamps are
// stored as UTC; the source zone is not used for clinical ordering.
func Timestamp(raw string) *time.Time {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		utc := parsed.UTC()
		return &utc
	}
	match := timestampHead.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	parts := []string{match[1], match[2], match[3], "00", "00", "00"}
	for i := 4; i <= 6; i++ {
		if match[i] != "" {
			parts[i-1] = match[i]
		}
	}
	parsed, err := time.Parse("20060102150405", strings.Join(parts, ""))
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}
