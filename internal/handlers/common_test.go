package handlers

import "testing"

func TestParsePositiveInt(t *testing.T) {
	if got := parsePositiveInt("5", 1); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
	if got := parsePositiveInt("0", 9); got != 9 {
		t.Fatalf("expected fallback 9, got %d", got)
	}
	if got := parsePositiveInt("abc", 7); got != 7 {
		t.Fatalf("expected fallback 7, got %d", got)
	}
}
