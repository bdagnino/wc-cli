package provider

import (
	"sort"
	"strings"
)

// TeamMatch is a fuzzy lookup result, ordered best-first by Score.
type TeamMatch struct {
	Team  Team
	Score int // higher is better
}

// FindTeams returns teams matching a free-form query, ranked best-first.
// It accepts exact abbreviations ("BRA"), full or partial names ("brazil",
// "bra"), and is case- and accent-insensitive for common cases. An empty
// result means nothing plausibly matched.
func FindTeams(teams []Team, query string) []TeamMatch {
	q := normalize(query)
	if q == "" {
		return nil
	}

	var out []TeamMatch
	for _, t := range teams {
		abbr := normalize(t.Abbr)
		name := normalize(t.Name)

		score := 0
		switch {
		case abbr == q || name == q:
			score = 100
		case strings.HasPrefix(abbr, q):
			score = 80
		case strings.HasPrefix(name, q):
			score = 70
		case strings.Contains(name, q):
			score = 50
		case strings.Contains(abbr, q):
			score = 40
		}
		if score > 0 {
			out = append(out, TeamMatch{Team: t, Score: score})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Team.Name < out[j].Team.Name
	})
	return out
}

// FindTeam returns the single best match, or ok=false if there is no match.
func FindTeam(teams []Team, query string) (Team, bool) {
	m := FindTeams(teams, query)
	if len(m) == 0 {
		return Team{}, false
	}
	return m[0].Team, true
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Fold a few common accented characters so "curacao" matches "Curaçao".
	repl := strings.NewReplacer(
		"ç", "c", "ã", "a", "á", "a", "à", "a", "â", "a",
		"é", "e", "è", "e", "ê", "e", "í", "i", "ó", "o",
		"ô", "o", "ú", "u", "ü", "u", "ñ", "n", "ş", "s",
		"ı", "i", "ğ", "g",
	)
	return repl.Replace(s)
}
