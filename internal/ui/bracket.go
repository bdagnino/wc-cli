package ui

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
)

// This file turns the flat list of knockout fixtures into a real bracket tree
// and draws it. ESPN hands us every knockout match up front — even the empty
// ones — and encodes the structure in the placeholder names: an empty Round of
// 16 slot is literally named "Round of 32 1 Winner", which tells us exactly
// which earlier match feeds it. So the whole tree (and the third-place game,
// whose slots are "...Loser") comes straight from the data, no hardcoding.

type bRound int

const (
	rR32 bRound = iota
	rR16
	rQF
	rSF
	rFinal
	nRounds
)

var roundTitles = [nRounds]string{"Round of 32", "Round of 16", "Quarterfinals", "Semifinals", "Final"}

// bSlot is one side of a match: a qualified team, a group placeholder
// ("Group I Winner"), or the winner/loser of an earlier match.
type bSlot struct {
	abbr string
	name string
	real bool // a qualified national team (not a placeholder)

	hasSrc   bool // filled by the winner of an earlier match
	srcRound bRound
	srcN     int
	loser    bool // references a loser (only the third-place game does)
}

type bMatch struct {
	round  bRound
	n      int    // 1-based position within the round, by ascending match id
	id     int    // numeric ESPN match id; the bracket's "match N" numbering
	kick   time.Time
	state  provider.MatchState
	home   bSlot
	away   bSlot
	hScore int
	aScore int

	upper *bMatch // feeder into home
	lower *bMatch // feeder into away

	center float64 // vertical center row, for layout
}

// Bracket is the parsed knockout tree.
type Bracket struct {
	rounds [nRounds][]*bMatch
	final  *bMatch
}

var refRe = regexp.MustCompile(`(?i)(round of 32|round of 16|quarterfinal|semifinal)\s+(\d+)\s+(winner|loser)`)

func refRound(s string) bRound {
	switch strings.ToLower(s) {
	case "round of 32":
		return rR32
	case "round of 16":
		return rR16
	case "quarterfinal":
		return rQF
	default:
		return rSF
	}
}

func classifyRound(round string) (bRound, bool) {
	r := strings.ToLower(round)
	switch {
	case strings.Contains(r, "32"):
		return rR32, true
	case strings.Contains(r, "16"):
		return rR16, true
	case strings.Contains(r, "quarter"):
		return rQF, true
	case strings.Contains(r, "semi"):
		return rSF, true
	case strings.Contains(r, "final"):
		return rFinal, true
	}
	return 0, false
}

func slotOf(t provider.Team) bSlot {
	if m := refRe.FindStringSubmatch(t.Name); m != nil {
		n, _ := strconv.Atoi(m[2])
		return bSlot{abbr: t.Abbr, name: t.Name, hasSrc: true,
			srcRound: refRound(m[1]), srcN: n, loser: strings.EqualFold(m[3], "loser")}
	}
	lower := strings.ToLower(t.Name)
	placeholder := t.Abbr == "" || strings.Contains(t.Name, "Group") || strings.Contains(lower, "place")
	return bSlot{abbr: t.Abbr, name: t.Name, real: !placeholder}
}

// BuildBracket parses knockout matches into a tree. ok is false when the feed
// doesn't carry a recognizable bracket yet (e.g. before the knockouts exist).
func BuildBracket(ms []provider.Match) (b *Bracket, ok bool) {
	b = &Bracket{}
	for _, m := range ms {
		r, valid := classifyRound(m.Round)
		if !valid {
			continue
		}
		bm := &bMatch{round: r, id: numericID(m.ID), kick: m.Kick, state: m.State,
			home: slotOf(m.Home), away: slotOf(m.Away), hScore: m.HomeScore, aScore: m.AwayScore}
		// The third-place game shares the "Semifinals" label but feeds off
		// losers — keep it out of the main tree.
		if bm.home.loser || bm.away.loser {
			continue
		}
		b.rounds[r] = append(b.rounds[r], bm)
	}
	// Number matches within each round by ascending match id — that is the
	// numbering ESPN's "Round of 32 N Winner" references point at (the ids are
	// assigned in bracket order, which is not the same as kickoff order) — then
	// resolve those references into real tree edges.
	for r := bRound(0); r < nRounds; r++ {
		sort.SliceStable(b.rounds[r], func(i, j int) bool {
			return b.rounds[r][i].id < b.rounds[r][j].id
		})
		for i, m := range b.rounds[r] {
			m.n = i + 1
		}
	}
	resolve := func(s bSlot) *bMatch {
		if !s.hasSrc || s.srcN < 1 || int(s.srcRound) >= len(roundTitles) {
			return nil
		}
		src := b.rounds[s.srcRound]
		if s.srcN-1 >= len(src) {
			return nil
		}
		return src[s.srcN-1]
	}
	for r := bRound(0); r < nRounds; r++ {
		for _, m := range b.rounds[r] {
			m.upper = resolve(m.home)
			m.lower = resolve(m.away)
		}
	}
	if len(b.rounds[rFinal]) == 1 && len(b.rounds[rR32]) > 0 {
		b.final = b.rounds[rFinal][0]
		return b, true
	}
	return b, false
}

// IDs are numeric strings; sort.SliceStable on kick handles ordering, but keep a
// numeric helper in case two kickoffs tie.
func numericID(id string) int { n, _ := strconv.Atoi(id); return n }
