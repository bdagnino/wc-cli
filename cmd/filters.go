package cmd

import (
	"strings"
	"time"

	"github.com/bdagnino/wcup/internal/provider"
)

// filterOpts holds the shared filters used by schedule/results/next.
type filterOpts struct {
	team  string
	group string
	date  string
	round string
	limit int
}

// apply narrows matches by the configured filters, preserving order.
func (f filterOpts) apply(matches []provider.Match, loc *time.Location) []provider.Match {
	var out []provider.Match
	for _, m := range matches {
		if f.team != "" && !matchHasTeam(m, f.team) {
			continue
		}
		if f.group != "" && !strings.EqualFold(m.Group, strings.TrimSpace(f.group)) {
			continue
		}
		if f.round != "" && !roundMatches(m.Round, f.round) {
			continue
		}
		if f.date != "" {
			want, ok := parseDate(f.date, loc)
			if ok && !sameDay(m.Kick.In(loc), want) {
				continue
			}
		}
		out = append(out, m)
		if f.limit > 0 && len(out) >= f.limit {
			break
		}
	}
	return out
}

func matchHasTeam(m provider.Match, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	for _, s := range []string{m.Home.Abbr, m.Home.Name, m.Away.Abbr, m.Away.Name} {
		if strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}

// roundMatches maps short tokens (group, r32, r16, qf, sf, final) to ESPN's
// human round names.
func roundMatches(roundName, token string) bool {
	r := strings.ToLower(roundName)
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "group", "groups", "gs":
		return strings.Contains(r, "group")
	case "r32", "ro32", "round32":
		return strings.Contains(r, "32")
	case "r16", "ro16", "round16":
		return strings.Contains(r, "16")
	case "qf", "quarter", "quarterfinals":
		return strings.Contains(r, "quarter")
	case "sf", "semi", "semifinals":
		return strings.Contains(r, "semi")
	case "final", "f":
		return strings.Contains(r, "final") && !strings.Contains(r, "semi") && !strings.Contains(r, "quarter")
	default:
		return strings.Contains(r, strings.ToLower(token))
	}
}

// parseDate accepts "today", "tomorrow", "yesterday", or YYYY-MM-DD.
func parseDate(s string, loc *time.Location) (time.Time, bool) {
	now := time.Now().In(loc)
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today":
		return now, true
	case "tomorrow":
		return now.AddDate(0, 0, 1), true
	case "yesterday":
		return now.AddDate(0, 0, -1), true
	}
	if t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), loc); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
