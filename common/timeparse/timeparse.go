package timeparse

import "time"

// LayoutDate is the standard YYYY-MM-DD date layout.
const LayoutDate = "2006-01-02"

// ParseOptional parses a layout-formatted date string pointer.
// Nil or empty input returns (nil, true). Parse failure returns (nil, false).
// Success returns (&t, true).
func ParseOptional(value *string, layout string) (*time.Time, bool) {
	if value == nil || *value == "" {
		return nil, true
	}
	t, err := time.Parse(layout, *value)
	if err != nil {
		return nil, false
	}
	return &t, true
}

// ParseStrict parses value using layout and round-trips the result to ensure
// the input is canonical (e.g. "2024-1-5" parses but fails because it formats
// back as "2024-01-05"). Returns (nil, false) on any mismatch.
func ParseStrict(value, layout string) (*time.Time, bool) {
	t, err := time.Parse(layout, value)
	if err != nil || t.Format(layout) != value {
		return nil, false
	}
	return &t, true
}
