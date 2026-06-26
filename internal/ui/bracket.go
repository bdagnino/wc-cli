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

	projected bool // penciled in from current group standings, not yet qualified

	hasSrc   bool // filled by the winner of an earlier match
	srcRound bRound
	srcN     int
	loser    bool // references a loser (only the third-place game does)
}

type bMatch struct {
	round  bRound
	n      int    // 1-based position within the round, by bracket match number
	num    int    // source bracket position (FIFA match number); orders the tree
	id     int    // numeric ESPN match id; fallback ordering when num is absent
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
		bm := &bMatch{round: r, num: m.MatchNumber, id: numericID(m.ID), kick: m.Kick, state: m.State,
			home: slotOf(m.Home), away: slotOf(m.Away), hScore: m.HomeScore, aScore: m.AwayScore}
		// The third-place game shares the "Semifinals" label but feeds off
		// losers — keep it out of the main tree.
		if bm.home.loser || bm.away.loser {
			continue
		}
		b.rounds[r] = append(b.rounds[r], bm)
	}
	// Number matches within each round by ascending bracket match number —
	// that is the numbering ESPN's "Round of 32 N Winner" references point at.
	// Event ids are NOT in bracket order (ESPN assigns them in a different
	// permutation), so we order by the match number fetched from the feed and
	// only fall back to id when it's missing — then resolve those references
	// into real tree edges.
	for r := bRound(0); r < nRounds; r++ {
		sort.SliceStable(b.rounds[r], func(i, j int) bool {
			a, c := b.rounds[r][i], b.rounds[r][j]
			if a.num > 0 && c.num > 0 {
				return a.num < c.num
			}
			return a.id < c.id
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

// slotTokenRe matches a group-placeholder code like "1A", "2B" or "3I":
// rank 1–3 followed by a group letter A–L.
var slotTokenRe = regexp.MustCompile(`^([1-3])([A-La-l])$`)

// slotNameRe pulls the group letter out of a placeholder name like
// "Group A Winner" or "Group I 3rd Place".
var slotNameRe = regexp.MustCompile(`(?i)group\s+([A-L])\b`)

// multiGroupRe spots an indeterminate third-place slot whose name lists several
// groups, e.g. "Third Place Group A/B/C/D/F". Which group's third-placed team
// actually lands there isn't settled by the standings alone (it follows FIFA's
// best-thirds allocation table), so these are never penciled in.
var multiGroupRe = regexp.MustCompile(`(?i)group\s+[A-L]\s*/`)

// groupSlotRef reports the group letter and finishing rank a placeholder slot
// stands for ("1A" → A, 1). It prefers the short code and falls back to parsing
// the name, so it works whether ESPN sends "2C" or "Group C Runner-Up".
func groupSlotRef(s bSlot) (rank int, letter string, ok bool) {
	if m := slotTokenRe.FindStringSubmatch(strings.TrimSpace(s.abbr)); m != nil {
		rank, _ = strconv.Atoi(m[1])
		return rank, strings.ToUpper(m[2]), true
	}
	if multiGroupRe.MatchString(s.name) {
		return 0, "", false
	}
	lm := slotNameRe.FindStringSubmatch(s.name)
	if lm == nil {
		return 0, "", false
	}
	letter = strings.ToUpper(lm[1])
	low := strings.ToLower(s.name)
	switch {
	case strings.Contains(low, "winner") || strings.Contains(low, "1st") || strings.Contains(low, "first"):
		rank = 1
	case strings.Contains(low, "runner") || strings.Contains(low, "2nd") || strings.Contains(low, "second"):
		rank = 2
	case strings.Contains(low, "3rd") || strings.Contains(low, "third"):
		rank = 3
	default:
		return 0, "", false
	}
	return rank, letter, true
}

// standingsTeam returns the team currently sitting at the given rank in the
// given group, by rank field with a slice-position fallback.
func standingsTeam(groups []provider.Group, letter string, rank int) (provider.Team, bool) {
	for _, g := range groups {
		if !strings.EqualFold(g.Letter, letter) {
			continue
		}
		for _, st := range g.Standings {
			if st.Rank == rank {
				return st.Team, true
			}
		}
		if rank >= 1 && rank-1 < len(g.Standings) {
			return g.Standings[rank-1].Team, true
		}
	}
	return provider.Team{}, false
}

// Project pencils in group-placeholder slots (1A, 2B, 3I…) with the team that
// would occupy each if the current group standings held. Slots already filled
// by a qualified team or fed by an earlier match are left alone, as are ones
// ESPN hasn't pinned to a specific group and rank (e.g. a bare "3RD"). It
// returns how many slots it filled.
func (b *Bracket) Project(groups []provider.Group) int {
	n := 0
	for r := bRound(0); r < nRounds; r++ {
		for _, m := range b.rounds[r] {
			if projectSlot(&m.home, groups) {
				n++
			}
			if projectSlot(&m.away, groups) {
				n++
			}
		}
	}
	return n
}

func projectSlot(s *bSlot, groups []provider.Group) bool {
	if s.real || s.hasSrc || s.projected {
		return false
	}
	rank, letter, ok := groupSlotRef(*s)
	if !ok {
		return false
	}
	t, ok := standingsTeam(groups, letter, rank)
	if !ok || t.Abbr == "" {
		return false
	}
	s.abbr = t.Abbr
	s.name = t.Name
	s.projected = true
	return true
}
