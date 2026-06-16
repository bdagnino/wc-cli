package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var matchCmd = &cobra.Command{
	Use:   "match [team]",
	Short: "Detail for a single match (live, featured, or by team)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}

		var chosen *provider.Match
		if len(args) > 0 {
			query := joinArgs(args)
			if isNumeric(query) {
				chosen = pickByID(all, query)
			} else {
				chosen = pickByTeam(all, query, loc)
			}
			if chosen == nil {
				fmt.Println(ui.Muted.Render("No match found for \"" + query + "\"."))
				return nil
			}
		} else {
			chosen = pickFeatured(all)
			if chosen == nil {
				fmt.Println(ui.Muted.Render("No matches available right now."))
				return nil
			}
		}

		// Enrich with timeline + game info. Best-effort: detail failures
		// shouldn't blank out the core scoreline.
		detail, derr := p.Detail(ctx, chosen.ID)
		detail.Match = *chosen

		if flagJSON {
			_, err := emitJSON(detail)
			return err
		}
		fmt.Print(renderMatchDetail(detail, loc))
		if derr != nil {
			fmt.Println(ui.Faint.Render("(timeline unavailable)"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(matchCmd)
}

// pickByTeam returns the most relevant match for a team: a live one, else the
// next upcoming, else the most recent finished.
func pickByTeam(all []provider.Match, query string, loc *time.Location) *provider.Match {
	var live, next, last *provider.Match
	now := time.Now()
	for i := range all {
		m := &all[i]
		if !matchHasTeam(*m, query) {
			continue
		}
		switch m.State {
		case provider.StateLive:
			live = m
		case provider.StateScheduled:
			if m.Kick.After(now) && next == nil {
				next = m
			}
		case provider.StateFinished:
			last = m
		}
	}
	switch {
	case live != nil:
		return live
	case next != nil:
		return next
	default:
		return last
	}
}

func pickByID(all []provider.Match, id string) *provider.Match {
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// pickFeatured returns the first live match, else the next upcoming overall.
func pickFeatured(all []provider.Match) *provider.Match {
	now := time.Now()
	for i := range all {
		if all[i].State == provider.StateLive {
			return &all[i]
		}
	}
	for i := range all {
		if all[i].State == provider.StateScheduled && all[i].Kick.After(now) {
			return &all[i]
		}
	}
	if len(all) > 0 {
		return &all[len(all)-1]
	}
	return nil
}

// stageLabel describes the round without redundancy: a group-stage match reads
// "Group J", a knockout match reads its round name (e.g. "Quarterfinals").
func stageLabel(m provider.Match) string {
	if m.Group != "" {
		return "Group " + m.Group
	}
	return m.Round
}

func renderMatchDetail(d provider.MatchDetail, loc *time.Location) string {
	m := d.Match
	var status string
	switch m.State {
	case provider.StateLive:
		status = ui.Live.Render("● LIVE " + m.Clock)
	case provider.StateFinished:
		status = ui.Muted.Render("Full time")
	default:
		status = ui.Upcoming.Render(m.Kick.In(loc).Format("Mon, Jan 2 · 15:04"))
	}

	scoreline := ui.Flag(m.Home.Abbr) + " " + ui.Header.Render(m.Home.Name)
	if m.State == provider.StateScheduled {
		scoreline += ui.Faint.Render("  vs  ")
	} else {
		scoreline += ui.Score.Render(fmt.Sprintf("  %d – %d  ", m.HomeScore, m.AwayScore))
	}
	scoreline += ui.Header.Render(m.Away.Name) + " " + ui.Flag(m.Away.Abbr)

	out := scoreline + "\n" + status + "\n"
	if stage := stageLabel(m); stage != "" {
		out += ui.Faint.Render(stage) + "\n"
	}
	if m.Venue != "" {
		venue := "📍 " + m.Venue
		if d.Attendance > 0 {
			venue += fmt.Sprintf("  ·  %s attendance", humanInt(d.Attendance))
		}
		out += ui.Faint.Render(venue) + "\n"
	}
	if len(d.Events) > 0 {
		out += "\n" + ui.Timeline(d.Events)
	}
	return out
}

// humanInt formats an integer with thousands separators (80824 -> "80,824").
func humanInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
