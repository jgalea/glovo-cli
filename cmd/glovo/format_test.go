package main

import (
	"strings"
	"testing"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func TestBasketTextEmpty(t *testing.T) {
	b := &glovo.Basket{}
	if got := basketText(b); got != "(empty)" {
		t.Fatalf("basketText empty: got %q, want %q", got, "(empty)")
	}
}

func TestBasketTextWithLines(t *testing.T) {
	b := &glovo.Basket{
		Currency: "EUR",
		Total:    12.5,
		Lines: []glovo.BasketLine{
			{ProductID: 42, Name: "Boscaiola", Qty: 2, UnitPrice: 6.25},
		},
	}
	got := basketText(b)
	if !strings.Contains(got, "2x [42] Boscaiola") {
		t.Fatalf("basketText missing line: %q", got)
	}
	if !strings.Contains(got, "Total: 12.50") {
		t.Fatalf("basketText missing total: %q", got)
	}
}

func TestBasketTextLineWithoutNameHasSingleSeparator(t *testing.T) {
	b := &glovo.Basket{
		Currency: "EUR",
		Lines: []glovo.BasketLine{
			{ProductID: 7, Qty: 1, UnitPrice: 6.25},
		},
	}
	got := basketText(b)
	if strings.Contains(got, "]  —") {
		t.Fatalf("basketText has a double space before the dash for an unnamed line: %q", got)
	}
	if !strings.Contains(got, "1x [7] — 6.25") {
		t.Fatalf("basketText missing single-separator line: %q", got)
	}
}
