package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/charmbracelet/lipgloss"
)

// nameW fits the longest real team name ("Bosnia-Herzegovina", 18). Inline +
// MaxWidth keep every cell on one line: short names pad, long placeholder names
// (e.g. "Group A 2nd Place" in the bracket) truncate instead of wrapping.
const nameW = 18

var (
	homeCell = lipgloss.NewStyle().Width(nameW).MaxWidth(nameW).Inline(true).Align(lipgloss.Right)
	awayCell = lipgloss.NewStyle().Width(nameW).MaxWidth(nameW).Inline(true).Align(lipgloss.Left)
)

// Match renders a single match as one aligned line. anchor is the day the list
// is "about" (e.g. today): when a scheduled kickoff's local clock time falls on
// a different calendar day, the time gets a flight-style "+1" / "-1" marker.
// Pass the zero time to suppress the marker (e.g. in day-grouped lists).
func Match(m provider.Match, loc *time.Location, anchor time.Time) string {
	var status string
	switch m.State {
	case provider.StateLive:
		clock := m.Clock
		if clock == "" {
			clock = "LIVE"
		}
		status = Live.Render("● " + clock)
	case provider.StateFinished:
		status = Muted.Render("FT")
	default:
		kick := m.Kick.In(loc)
		status = Upcoming.Render(kick.Format("15:04"))
		if d := dayDelta(anchor, kick, loc); d != 0 {
			status += Faint.Render(fmt.Sprintf(" %+d", d))
		}
	}

	home := homeCell.Render(m.Home.Name)
	away := awayCell.Render(m.Away.Name)

	var mid string
	if m.State == provider.StateScheduled {
		mid = Faint.Render("  vs  ")
	} else {
		hs, as := fmt.Sprintf("%d", m.HomeScore), fmt.Sprintf("%d", m.AwayScore)
		hStyle, aStyle := Score, Score
		if m.State == provider.StateFinished {
			switch {
			case m.HomeScore > m.AwayScore:
				aStyle = Muted
			case m.AwayScore > m.HomeScore:
				hStyle = Muted
			}
		}
		mid = " " + hStyle.Render(hs) + Faint.Render(" - ") + aStyle.Render(as) + " "
	}

	meta := ""
	if m.Group != "" {
		meta = Faint.Render("  Grp " + m.Group)
	} else if m.Round != "" {
		meta = Faint.Render("  " + m.Round)
	}

	return fmt.Sprintf("%s %s %s %s %s %s%s",
		statusPad(status),
		Flag(m.Home.Abbr), home,
		mid,
		away, Flag(m.Away.Abbr),
		meta,
	)
}

// dayDelta reports how many calendar days kick lands after anchor, both read in
// loc. It is 0 when anchor is unset or the two share a day. This lets a flat
// list (which prints only a clock time, no date) flag a kickoff whose local
// time has rolled past midnight — e.g. a match ESPN buckets under "today" by US
// date but that actually starts 03:00 the next morning here — with a "+1".
func dayDelta(anchor, kick time.Time, loc *time.Location) int {
	if anchor.IsZero() {
		return 0
	}
	midnight := func(t time.Time) time.Time {
		t = t.In(loc)
		y, mo, d := t.Date()
		return time.Date(y, mo, d, 0, 0, 0, 0, loc)
	}
	// Round to absorb DST days that are 23 or 25 hours long.
	return int(math.Round(midnight(kick).Sub(midnight(anchor)).Hours() / 24))
}

// statusPad keeps the status column visually aligned across styled strings,
// where the rendered width differs from the byte length.
func statusPad(s string) string {
	const target = 8
	pad := target - lipgloss.Width(s)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

// MatchList renders a titled block of matches grouped under a heading. anchor
// is the day the list is about (e.g. today); kickoffs on another calendar day
// get a "+1"-style marker. Pass the zero time to suppress it.
func MatchList(title string, matches []provider.Match, loc *time.Location, anchor time.Time) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(Title.Render(title) + "\n")
	}
	if len(matches) == 0 {
		b.WriteString(Muted.Render("  (none)") + "\n")
		return b.String()
	}
	for _, m := range matches {
		b.WriteString("  " + Match(m, loc, anchor) + "\n")
	}
	return b.String()
}

// MatchListByDay renders matches grouped by calendar day in the given location.
func MatchListByDay(title string, matches []provider.Match, loc *time.Location) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(Title.Render(title) + "\n")
	}
	if len(matches) == 0 {
		b.WriteString(Muted.Render("  (none)") + "\n")
		return b.String()
	}
	var lastDay string
	for _, m := range matches {
		day := m.Kick.In(loc).Format("Mon, Jan 2")
		if day != lastDay {
			b.WriteString("\n" + Header.Render(day) + "\n")
			lastDay = day
		}
		// Day headers already disambiguate the date, so no per-row marker.
		b.WriteString("  " + Match(m, loc, time.Time{}) + "\n")
	}
	return b.String()
}

// Standings renders group tables. If filter is non-empty, only that group letter.
func Standings(groups []provider.Group, filter string) string {
	var b strings.Builder
	filter = strings.ToUpper(strings.TrimSpace(filter))
	for _, g := range groups {
		if filter != "" && g.Letter != filter {
			continue
		}
		b.WriteString(Title.Render("Group "+g.Letter) + "\n")
		b.WriteString(Faint.Render(fmt.Sprintf("  %-18s %3s %3s %3s %3s %4s %4s", "", "P", "W", "D", "L", "GD", "Pts")) + "\n")
		for i, s := range g.Standings {
			row := fmt.Sprintf("%s %-16s %3d %3d %3d %3d %+4d %4d",
				Flag(s.Team.Abbr), s.Team.Name, s.Played, s.Won, s.Drawn, s.Lost, s.GoalDiff, s.Points)
			// Top two advance: highlight them.
			if i < 2 {
				row = Winner.Render(row)
			} else {
				row = Muted.Render(row)
			}
			b.WriteString("  " + row + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Teams renders the team list grouped by group letter, or flat alphabetical.
func Teams(teams []provider.Team, flat bool) string {
	var b strings.Builder
	if flat {
		for _, t := range teams {
			b.WriteString(fmt.Sprintf("  %s %s %s\n", Flag(t.Abbr), Header.Render(t.Abbr), t.Name))
		}
		return b.String()
	}

	byGroup := map[string][]provider.Team{}
	var order []string
	ungrouped := []provider.Team{}
	for _, t := range teams {
		if t.Group == "" {
			ungrouped = append(ungrouped, t)
			continue
		}
		if _, ok := byGroup[t.Group]; !ok {
			order = append(order, t.Group)
		}
		byGroup[t.Group] = append(byGroup[t.Group], t)
	}
	sortStrings(order)
	for _, g := range order {
		b.WriteString(Title.Render("Group "+g) + "\n")
		for _, t := range byGroup[g] {
			b.WriteString(fmt.Sprintf("  %s %s  %s\n", Flag(t.Abbr), Header.Render(t.Abbr), t.Name))
		}
		b.WriteString("\n")
	}
	if len(ungrouped) > 0 {
		b.WriteString(Title.Render("Teams") + "\n")
		for _, t := range ungrouped {
			b.WriteString(fmt.Sprintf("  %s %s  %s\n", Flag(t.Abbr), Header.Render(t.Abbr), t.Name))
		}
	}
	return b.String()
}

// Timeline renders a match's key events (goals, cards, subs) as one line each.
func Timeline(events []provider.MatchEvent) string {
	var b strings.Builder
	b.WriteString(Title.Render("Timeline") + "\n")
	for _, e := range events {
		icon, label := eventGlyph(e.Kind)
		clock := e.Clock
		if clock == "" {
			clock = "·"
		}
		line := fmt.Sprintf("  %-5s %s ", Faint.Render(clock), icon)
		line += eventBody(e, label)
		b.WriteString(line + "\n")
	}
	return b.String()
}

// eventBody describes who was involved, formatted per event kind.
func eventBody(e provider.MatchEvent, label string) string {
	if len(e.Players) == 0 {
		return Muted.Render(e.Text)
	}
	switch e.Kind {
	case provider.EventGoal, provider.EventPenalty, provider.EventOwnGoal:
		s := Header.Render(e.Players[0]) + Faint.Render("  "+label)
		if len(e.Players) > 1 {
			s += Faint.Render(" (assist: " + e.Players[1] + ")")
		}
		return s
	case provider.EventSub:
		if len(e.Players) > 1 {
			return Muted.Render(e.Players[0]) + Faint.Render(" for ") + Muted.Render(e.Players[1])
		}
		return Muted.Render(e.Players[0])
	default: // cards
		return Header.Render(e.Players[0]) + Faint.Render("  "+label)
	}
}

// eventGlyph maps an event kind to an icon and a short label.
func eventGlyph(k provider.EventKind) (icon, label string) {
	switch k {
	case provider.EventGoal:
		return "⚽", "goal"
	case provider.EventOwnGoal:
		return "⚽", "own goal"
	case provider.EventPenalty:
		return "⚽", "penalty"
	case provider.EventYellow:
		return "🟨", "yellow card"
	case provider.EventRed:
		return "🟥", "red card"
	case provider.EventSub:
		return "🔁", "substitution"
	default:
		return "•", ""
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
