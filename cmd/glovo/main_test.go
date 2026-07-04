package main

import "testing"

func TestRunVersion(t *testing.T) {
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("version exit = %d, want 0", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if code := run([]string{"nope"}); code != 2 {
		t.Fatalf("unknown exit = %d, want 2", code)
	}
}
