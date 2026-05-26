package main

import "testing"

func TestNormalizeProcessStatusUsesTopLetters(t *testing.T) {
	tests := map[string]string{
		"running":  "R",
		"sleep":    "S",
		"sleeping": "S",
		"zombie":   "Z",
		"stopped":  "T",
		"D":        "D",
		"unknown":  "",
	}

	for input, want := range tests {
		if got := normalizeProcessStatus(input); got != want {
			t.Fatalf("normalizeProcessStatus(%q) = %q, want %q", input, got, want)
		}
	}
}
