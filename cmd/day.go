package cmd

import (
	"fmt"
	"time"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var dayWatch bool

var dayCmd = &cobra.Command{
	Use:   "day [when]",
	Short: "Matches for a given day (defaults to today)",
	Long: "Show a single day's matches with live scores.\n\n" +
		"With no argument it's today (same as `wcup today`); otherwise pass the day:\n" +
		"  wcup day yesterday\n" +
		"  wcup day tomorrow\n" +
		"  wcup day 2026-06-25",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		when := ""
		if len(args) == 1 {
			when = args[0]
		}
		return runDayView(dayWatch, when)
	},
}

// runDayView fetches and renders one day's scoreboard. when is "" for today, or
// a value parseDate understands (yesterday/today/tomorrow/YYYY-MM-DD).
func runDayView(watch bool, when string) error {
	ctx, p, loc := setup()
	// Resolve a fixed day up front so a bad value errors before any fetch.
	var fixed time.Time
	if when != "" {
		d, ok := parseDate(when, loc)
		if !ok {
			return fmt.Errorf("invalid day %q: use yesterday, today, tomorrow or YYYY-MM-DD", when)
		}
		fixed = d
	}
	return runWatch(watch, func() (string, error) {
		now := time.Now().In(loc)
		day := now // when=="" re-reads each tick so --watch rolls over at midnight
		if when != "" {
			day = fixed
		}
		matches, err := p.Scoreboard(ctx, day)
		if err != nil {
			return "", err
		}
		if done, err := emitJSON(localize(matches, loc)); done || err != nil {
			return "", err
		}
		return ui.MatchList(dayTitle(day, now), matches, loc, day), nil
	})
}

// dayTitle labels a day relative to now — Today/Yesterday/Tomorrow — else just
// the date, e.g. "Today · Mon, Jun 23" or "Thu, Jun 25".
func dayTitle(day, now time.Time) string {
	date := day.Format("Mon, Jan 2")
	switch {
	case sameDay(day, now):
		return "Today · " + date
	case sameDay(day, now.AddDate(0, 0, -1)):
		return "Yesterday · " + date
	case sameDay(day, now.AddDate(0, 0, 1)):
		return "Tomorrow · " + date
	}
	return date
}

func init() {
	dayCmd.Flags().BoolVarP(&dayWatch, "watch", "w", false, "auto-refresh in place")
	rootCmd.AddCommand(dayCmd)
}
