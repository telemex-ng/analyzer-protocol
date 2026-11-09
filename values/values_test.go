package values

import (
	"testing"
	"time"
)

func TestNumeric(t *testing.T) {
	cases := map[string]float64{
		"9.4":     9.4,
		"<0.5":    0.5,
		"> 100":   100,
		"1,234.5": 1234.5,
	}
	for raw, want := range cases {
		got := Numeric(raw)
		if got == nil || *got != want {
			t.Fatalf("Numeric(%q) = %v, want %v", raw, got, want)
		}
	}
	for _, raw := range []string{"", "NEG", "POSITIVE", "abc"} {
		if got := Numeric(raw); got != nil {
			t.Fatalf("Numeric(%q) = %v, want nil", raw, got)
		}
	}
}

func TestRange(t *testing.T) {
	low, high := Range("4.0-10.0")
	if low == nil || high == nil || *low != 4 || *high != 10 {
		t.Fatalf("Range = %v, %v", low, high)
	}
	low, high = Range("<5")
	if low != nil || high == nil || *high != 5 {
		t.Fatalf("Range(<5) = %v, %v", low, high)
	}
	low, high = Range(">2")
	if low == nil || *low != 2 || high != nil {
		t.Fatalf("Range(>2) = %v, %v", low, high)
	}
	low, high = Range("4.0 - 10.0")
	if low == nil || high == nil || *low != 4 || *high != 10 {
		t.Fatalf("Range spaced = %v, %v", low, high)
	}
	if low, high := Range("negative"); low != nil || high != nil {
		t.Fatalf("Range(negative) = %v, %v", low, high)
	}
}

func TestTimestamp(t *testing.T) {
	got := Timestamp("20261108120000")
	if got == nil || !got.Equal(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("Timestamp = %v", got)
	}
	if got := Timestamp("not-a-time"); got != nil {
		t.Fatalf("Timestamp(not-a-time) = %v", got)
	}
}
