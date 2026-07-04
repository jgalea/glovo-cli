package main

import (
	"math"
	"testing"
)

func TestResolveCoordFlagWins(t *testing.T) {
	if got := resolveCoord(10.5, "GLOVO_TEST_LAT_UNSET", 41.3874); got != 10.5 {
		t.Fatalf("resolveCoord flag priority: got %v, want 10.5", got)
	}
}

func TestResolveCoordEnvFallback(t *testing.T) {
	t.Setenv("GLOVO_TEST_LAT", "48.8566")
	got := resolveCoord(math.NaN(), "GLOVO_TEST_LAT", 41.3874)
	if got != 48.8566 {
		t.Fatalf("resolveCoord env fallback: got %v, want 48.8566", got)
	}
}

func TestResolveCoordDefault(t *testing.T) {
	got := resolveCoord(math.NaN(), "GLOVO_TEST_LAT_UNSET_TOO", 41.3874)
	if got != 41.3874 {
		t.Fatalf("resolveCoord default: got %v, want 41.3874", got)
	}
}
