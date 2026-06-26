package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
)

func km(id, round, hAbbr, hName, aAbbr, aName string, kick time.Time) provider.Match {
	return provider.Match{
		ID: id, Round: round, Kick: kick, State: provider.StateScheduled,
		Home: provider.Team{Abbr: hAbbr, Name: hName},
		Away: provider.Team{Abbr: aAbbr, Name: aName},
	}
}

// A 4-team mini bracket: 4 × R32 → 2 × R16 → Final, plus a third-place game.
// Kickoffs are deliberately in the opposite order to the ids, so the test
// fails if numbering ever falls back to kickoff order instead of match id.
func miniBracket() []provider.Match {
	t := func(d int) time.Time { return time.Date(2026, 7, d, 12, 0, 0, 0, time.UTC) }
	return []provider.Match{
		km("100", "Round of 32", "ARG", "Argentina", "BRA", "Brazil", t(9)),
		km("101", "Round of 32", "ESP", "Spain", "FRA", "France", t(8)),
		km("102", "Round of 32", "GER", "Germany", "ITA", "Italy", t(7)),
		km("103", "Round of 32", "NED", "Netherlands", "POR", "Portugal", t(6)),
		km("200", "Round of 16", "", "Round of 32 1 Winner", "", "Round of 32 2 Winner", t(12)),
		km("201", "Round of 16", "", "Round of 32 3 Winner", "", "Round of 32 4 Winner", t(13)),
		km("299", "Semifinals", "", "Semifinal 1 Loser", "", "Semifinal 2 Loser", t(14)),
		km("300", "Final", "", "Round of 16 1 Winner", "", "Round of 16 2 Winner", t(15)),
	}
}

func TestBuildBracketStructure(t *testing.T) {
	b, ok := BuildBracket(miniBracket())
	if !ok {
		t.Fatal("BuildBracket reported the tree is not formed")
	}
	if got := len(b.rounds[rR32]); got != 4 {
		t.Fatalf("R32 count = %d, want 4", got)
	}
	if got := len(b.rounds[rR16]); got != 2 {
		t.Fatalf("R16 count = %d, want 2", got)
	}
	// The third-place game (Semifinal losers) must be excluded entirely.
	if got := len(b.rounds[rSF]); got != 0 {
		t.Fatalf("SF count = %d, want 0 (third-place game should be dropped)", got)
	}
	if b.final == nil {
		t.Fatal("final is nil")
	}

	r16a, r16b := b.rounds[rR16][0], b.rounds[rR16][1]
	if b.final.upper != r16a || b.final.lower != r16b {
		t.Fatal("final is not fed by the two R16 matches")
	}
	// R16 #1 must be fed by R32 #1 and #2, numbered by ascending id even though
	// their kickoffs are latest.
	if r16a.upper != b.rounds[rR32][0] || r16a.lower != b.rounds[rR32][1] {
		t.Fatal("R16 #1 feeders resolved to the wrong R32 matches")
	}
	if r16b.upper != b.rounds[rR32][2] || r16b.lower != b.rounds[rR32][3] {
		t.Fatal("R16 #2 feeders resolved to the wrong R32 matches")
	}
	if b.rounds[rR32][0].home.abbr != "ARG" {
		t.Fatalf("R32 #1 home = %q, want ARG (id-ordered)", b.rounds[rR32][0].home.abbr)
	}
}

func TestBracketPath(t *testing.T) {
	b, _ := BuildBracket(miniBracket())
	out, ok := b.Path("arg", time.UTC)
	if !ok {
		t.Fatal("Path(arg) not found")
	}
	for _, want := range []string{"Argentina", "Round of 32", "Round of 16", "Final"} {
		if !strings.Contains(out, want) {
			t.Errorf("path output missing %q:\n%s", want, out)
		}
	}
	if _, ok := b.Path("nope", time.UTC); ok {
		t.Error("Path(nope) should not match any team")
	}
}
