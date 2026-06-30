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

// numberedBracket is the same 4-team shape, but the event ids are in a
// different permutation from the bracket match numbers — exactly how ESPN's
// feed behaves. Ordering must follow MatchNumber, not the id. If it followed
// the id, R16 #1 would pair ARG with ESP instead of ARG with GER.
func numberedBracket() []provider.Match {
	t := func(d int) time.Time { return time.Date(2026, 7, d, 12, 0, 0, 0, time.UTC) }
	withNum := func(m provider.Match, num int) provider.Match { m.MatchNumber = num; return m }
	return []provider.Match{
		withNum(km("100", "Round of 32", "ARG", "Argentina", "BRA", "Brazil", t(6)), 73),
		withNum(km("101", "Round of 32", "ESP", "Spain", "FRA", "France", t(7)), 75),
		withNum(km("102", "Round of 32", "GER", "Germany", "ITA", "Italy", t(8)), 74),
		withNum(km("103", "Round of 32", "NED", "Netherlands", "POR", "Portugal", t(9)), 76),
		withNum(km("200", "Round of 16", "", "Round of 32 1 Winner", "", "Round of 32 2 Winner", t(12)), 89),
		withNum(km("201", "Round of 16", "", "Round of 32 3 Winner", "", "Round of 32 4 Winner", t(13)), 90),
		withNum(km("300", "Final", "", "Round of 16 1 Winner", "", "Round of 16 2 Winner", t(15)), 104),
	}
}

func TestBuildBracketOrdersByMatchNumber(t *testing.T) {
	b, ok := BuildBracket(numberedBracket())
	if !ok {
		t.Fatal("BuildBracket reported the tree is not formed")
	}
	// Round must be ordered by match number (ARG 73, GER 74, ESP 75, NED 76),
	// not by event id (ARG 100, ESP 101, GER 102, NED 103).
	want := []string{"ARG", "GER", "ESP", "NED"}
	for i, w := range want {
		if got := b.rounds[rR32][i].home.abbr; got != w {
			t.Fatalf("R32 #%d home = %q, want %q (match-number order)", i+1, got, w)
		}
	}
	// "Round of 32 2 Winner" must resolve to GER (number 74), not ESP (id 101).
	if got := b.rounds[rR16][0].lower; got != b.rounds[rR32][1] || got.home.abbr != "GER" {
		t.Fatal("R16 #1 second feeder did not resolve to GER via match number")
	}
}

// When a Round of 32 match finishes, the source fills the Round of 16 slot with
// the real winner rather than a "Round of 32 N Winner" placeholder, which
// severs the tree edge. The completed match must be re-linked (by team), not
// orphaned and hidden.
func TestBuildBracketRelinksFinishedFeeders(t *testing.T) {
	tm := func(d int) time.Time { return time.Date(2026, 7, d, 12, 0, 0, 0, time.UTC) }
	finished := func(m provider.Match, hs, as int) provider.Match {
		m.State = provider.StateFinished
		m.HomeScore, m.AwayScore = hs, as
		return m
	}
	ms := []provider.Match{
		// R32 #1 is over (ARG beat BRA); R32 #2 hasn't kicked off.
		finished(km("100", "Round of 32", "ARG", "Argentina", "BRA", "Brazil", tm(6)), 2, 1),
		km("101", "Round of 32", "ESP", "Spain", "FRA", "France", tm(7)),
		// R16 #1: ARG already penciled in by the source; the other side is still
		// a placeholder pointing at R32 #2.
		km("200", "Round of 16", "ARG", "Argentina", "", "Round of 32 2 Winner", tm(12)),
		km("300", "Final", "", "Round of 16 1 Winner", "", "Round of 16 2 Winner", tm(15)),
	}
	b, ok := BuildBracket(ms)
	if !ok {
		t.Fatal("BuildBracket reported the tree is not formed")
	}
	r16 := b.rounds[rR16][0]
	// The finished feeder (ARG vs BRA) must be re-attached as the upper edge...
	if r16.upper != b.rounds[rR32][0] {
		t.Fatal("finished feeder was not re-linked to the Round of 16 slot it produced")
	}
	// ...while the still-pending side resolves through its placeholder.
	if r16.lower != b.rounds[rR32][1] {
		t.Fatal("placeholder feeder did not resolve to R32 #2")
	}
}

func TestWonByHonoursShootout(t *testing.T) {
	// 1-1 in regulation, away won on penalties: the flag, not the level score,
	// decides the winner.
	pens := &bMatch{state: provider.StateFinished, hScore: 1, aScore: 1, winner: "away", shootout: true, hPen: 3, aPen: 4}
	if pens.wonBy(true) {
		t.Fatal("home lost the shootout but was marked the winner")
	}
	if !pens.wonBy(false) {
		t.Fatal("away won the shootout but was not marked the winner")
	}
	// No winner flag: fall back to the regulation score.
	reg := &bMatch{state: provider.StateFinished, hScore: 2, aScore: 1}
	if !reg.wonBy(true) || reg.wonBy(false) {
		t.Fatal("regulation winner-by-score fallback is broken")
	}
	// Before full time nobody has won, regardless of the running score.
	live := &bMatch{state: provider.StateLive, hScore: 1, aScore: 0}
	if live.wonBy(true) || live.wonBy(false) {
		t.Fatal("a match in progress should have no winner yet")
	}
}

// projectableBracket has a Round of 32 made entirely of group placeholders, in
// both the "2A" short-code and "Group A Winner" name styles, so Project can be
// exercised against both. Later rounds feed off match winners (TBD).
func projectableBracket() []provider.Match {
	t := func(d int) time.Time { return time.Date(2026, 7, d, 12, 0, 0, 0, time.UTC) }
	return []provider.Match{
		km("100", "Round of 32", "1A", "Group A Winner", "2B", "Group B Runner-Up", t(9)),
		km("101", "Round of 32", "", "Group C Winner", "", "Group D 2nd Place", t(8)),
		km("102", "Round of 32", "3I", "Group I 3rd Place", "1E", "Group E Winner", t(7)),
		km("103", "Round of 32", "3RD", "Third Place Group A/B/C/D/F", "9Z", "Bogus Slot", t(6)),
		km("200", "Round of 16", "", "Round of 32 1 Winner", "", "Round of 32 2 Winner", t(12)),
		km("201", "Round of 16", "", "Round of 32 3 Winner", "", "Round of 32 4 Winner", t(13)),
		km("300", "Final", "", "Round of 16 1 Winner", "", "Round of 16 2 Winner", t(15)),
	}
}

func standing(letter, abbr, name string, rank int) provider.GroupStanding {
	return provider.GroupStanding{Team: provider.Team{Abbr: abbr, Name: name, Group: letter}, Rank: rank}
}

func projectGroups() []provider.Group {
	return []provider.Group{
		{Letter: "A", Standings: []provider.GroupStanding{standing("A", "ARG", "Argentina", 1), standing("A", "CRO", "Croatia", 2)}},
		{Letter: "B", Standings: []provider.GroupStanding{standing("B", "ENG", "England", 1), standing("B", "USA", "United States", 2)}},
		{Letter: "C", Standings: []provider.GroupStanding{standing("C", "FRA", "France", 1), standing("C", "MEX", "Mexico", 2)}},
		{Letter: "D", Standings: []provider.GroupStanding{standing("D", "BRA", "Brazil", 1), standing("D", "JPN", "Japan", 2)}},
		{Letter: "E", Standings: []provider.GroupStanding{standing("E", "ESP", "Spain", 1), standing("E", "GER", "Germany", 2)}},
		{Letter: "I", Standings: []provider.GroupStanding{standing("I", "POR", "Portugal", 1), standing("I", "NED", "Netherlands", 2), standing("I", "GHA", "Ghana", 3)}},
	}
}

func TestBracketProject(t *testing.T) {
	b, ok := BuildBracket(projectableBracket())
	if !ok {
		t.Fatal("BuildBracket reported the tree is not formed")
	}
	n := b.Project(projectGroups())
	// 1A, 2B, Group C Winner, Group D 2nd Place, 3I, 1E = 6 fillable slots.
	if n != 6 {
		t.Fatalf("Project filled %d slots, want 6", n)
	}

	r32 := b.rounds[rR32]
	// Short code "1A" → group A first place.
	if got := r32[0].home; !got.projected || got.abbr != "ARG" {
		t.Errorf("slot 1A = %+v, want projected ARG", got)
	}
	// Short code "2B" → group B runner-up.
	if got := r32[0].away; !got.projected || got.abbr != "USA" {
		t.Errorf("slot 2B = %+v, want projected USA", got)
	}
	// Name-only "Group C Winner" → group C first place.
	if got := r32[1].home; !got.projected || got.abbr != "FRA" {
		t.Errorf("slot Group C Winner = %+v, want projected FRA", got)
	}
	// Name-only "Group D 2nd Place" → group D runner-up.
	if got := r32[1].away; !got.projected || got.abbr != "JPN" {
		t.Errorf("slot Group D 2nd Place = %+v, want projected JPN", got)
	}
	// Third-place code "3I" → group I third place.
	if got := r32[2].home; !got.projected || got.abbr != "GHA" {
		t.Errorf("slot 3I = %+v, want projected GHA", got)
	}
	// Unresolvable placeholders stay put: an indeterminate multi-group third-place
	// slot ("Third Place Group A/B/C/D/F") and a bogus "9Z".
	if r32[3].home.projected || r32[3].away.projected {
		t.Errorf("indeterminate slots should not be projected: %+v / %+v", r32[3].home, r32[3].away)
	}

	// Penciled-in teams must not count as qualified, and match-fed slots stay TBD.
	if r32[0].home.real {
		t.Error("projected slot should not be marked real")
	}
	if b.rounds[rR16][0].home.projected {
		t.Error("a match-fed (hasSrc) slot must never be projected")
	}

	// The full render should now contain the penciled-in codes.
	out := b.Render(time.UTC)
	for _, want := range []string{"ARG", "USA", "FRA", "GHA"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered bracket missing projected %q", want)
		}
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
