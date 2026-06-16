package espn

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	cases := []string{
		"2026-06-16T19:00Z",    // ESPN's seconds-less form
		"2026-06-16T19:00:00Z", // with seconds
	}
	want := time.Date(2026, 6, 16, 19, 0, 0, 0, time.UTC)
	for _, c := range cases {
		got, err := parseTime(c)
		if err != nil {
			t.Errorf("parseTime(%q) errored: %v", c, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("parseTime(%q) = %v, want %v", c, got, want)
		}
	}
	if _, err := parseTime("not-a-time"); err == nil {
		t.Error("parseTime: expected error for invalid input")
	}
}

func TestGroupLetter(t *testing.T) {
	for in, want := range map[string]string{"Group A": "A", "Group L": "L", "": ""} {
		if got := groupLetter(in); got != want {
			t.Errorf("groupLetter(%q) = %q, want %q", in, got, want)
		}
	}
}
