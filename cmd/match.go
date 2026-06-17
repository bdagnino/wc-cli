package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bdagnino/wc-cli/internal/fifa"
	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	matchLast  bool
	matchVideo bool
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
				chosen = pickByTeam(all, query, loc, matchLast)
			}
			if chosen == nil {
				fmt.Println(ui.Muted.Render("No match found for \"" + query + "\"."))
				return nil
			}
		} else {
			chosen = pickFeatured(all, matchLast)
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
			detail.Kick = detail.Kick.In(loc)
			_, err := emitJSON(detail)
			return err
		}
		hl := matchHighlights(ctx, *chosen)
		fmt.Print(renderMatchDetail(detail, loc, hl))
		if derr != nil {
			fmt.Println(ui.Faint.Render("(timeline unavailable)"))
		}
		if matchVideo {
			openHighlights(hl)
		}
		return nil
	},
}

// openHighlights launches the highlights link in the default browser, or notes
// when there's nothing to open (a match that hasn't finished).
func openHighlights(url string) {
	if url == "" {
		fmt.Println(ui.Muted.Render("Nothing to open — highlights appear once a match has finished."))
		return
	}
	if err := openInBrowser(url); err != nil {
		fmt.Println(ui.Faint.Render("Couldn't open a browser: ") + err.Error())
		return
	}
	fmt.Println(ui.Faint.Render("Opening highlights in your browser…"))
}

func init() {
	matchCmd.Flags().BoolVar(&matchLast, "last", false, "show the most recent finished match (instead of live/next), e.g. wcup match arg --last")
	matchCmd.Flags().BoolVar(&matchVideo, "video", false, "open the highlights video in your default browser")
	rootCmd.AddCommand(matchCmd)
}

// pickByTeam returns the most relevant match for a team: a live one, else the
// next upcoming, else the most recent finished. When preferLast is set, the
// most recent finished match wins (falling back to live/next only if the team
// has none yet).
func pickByTeam(all []provider.Match, query string, loc *time.Location, preferLast bool) *provider.Match {
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
	case preferLast && last != nil:
		return last
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
// When preferLast is set, it returns the most recent finished match instead.
func pickFeatured(all []provider.Match, preferLast bool) *provider.Match {
	now := time.Now()
	if preferLast {
		// all is sorted ascending by kickoff, so the last finished entry is
		// the most recent one played.
		for i := len(all) - 1; i >= 0; i-- {
			if all[i].State == provider.StateFinished {
				return &all[i]
			}
		}
	}
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

// matchHighlights resolves a highlights link for a finished match: the
// official FIFA reel when it's published, otherwise a YouTube search that
// reliably surfaces it. Best-effort — a slow or failing FIFA lookup quietly
// falls back rather than stalling or erroring the detail view.
func matchHighlights(ctx context.Context, m provider.Match) string {
	if m.State != provider.StateFinished {
		return "" // highlights only exist once a match is done
	}
	fctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if u, err := fifa.New().Highlights(fctx, m.Home.Abbr, m.Away.Abbr, m.HomeScore, m.AwayScore); err == nil && u != "" {
		return u
	}
	return youtubeSearchURL(m)
}

func renderMatchDetail(d provider.MatchDetail, loc *time.Location, highlights string) string {
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
	if highlights != "" {
		out += ui.Faint.Render("▶ Highlights: ") + ui.Muted.Render(highlights) + "\n"
	}
	if len(d.Events) > 0 {
		out += "\n" + ui.Timeline(d.Events)
	}
	return out
}

// youtubeSearchURL builds a YouTube search link for a match's official
// highlights — the fallback when FIFA's exact link isn't available. It is
// deterministic from the team names, so it needs no network call.
func youtubeSearchURL(m provider.Match) string {
	q := fmt.Sprintf("%s vs %s highlights World Cup 2026", m.Home.Name, m.Away.Name)
	return "https://www.youtube.com/results?search_query=" + url.QueryEscape(q)
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
