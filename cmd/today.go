package cmd

import (
	"fmt"
	"time"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	todayWatch bool
	todayDate  string
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "A day's matches with live scores (defaults to today)",
	Long: "Show a single day's matches with live scores.\n\n" +
		"Defaults to today; pass --date to look up another day:\n" +
		"  wcup today\n" +
		"  wcup today --date yesterday\n" +
		"  wcup today --date tomorrow\n" +
		"  wcup today --date 2026-06-25",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		// Resolve --date up front so a bad value errors before any fetch.
		var fixed time.Time
		if todayDate != "" {
			d, ok := parseDate(todayDate, loc)
			if !ok {
				return fmt.Errorf("invalid --date %q: use today, yesterday, tomorrow or YYYY-MM-DD", todayDate)
			}
			fixed = d
		}
		return runWatch(todayWatch, func() (string, error) {
			now := time.Now().In(loc)
			day := fixed
			if todayDate == "" {
				day = now // re-read each tick so --watch rolls over at midnight
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
	},
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
	todayCmd.Flags().BoolVarP(&todayWatch, "watch", "w", false, "auto-refresh in place")
	todayCmd.Flags().StringVar(&todayDate, "date", "", "day to show: today, yesterday, tomorrow or YYYY-MM-DD")
	rootCmd.AddCommand(todayCmd)
}
