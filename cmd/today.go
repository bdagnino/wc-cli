package cmd

import (
	"time"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var todayWatch bool

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Today's matches with live scores",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		return runWatch(todayWatch, func() (string, error) {
			matches, err := p.Scoreboard(ctx, time.Now().In(loc))
			if err != nil {
				return "", err
			}
			if done, err := emitJSON(localize(matches, loc)); done || err != nil {
				return "", err
			}
			now := time.Now().In(loc)
			title := "Today · " + now.Format("Mon, Jan 2")
			return ui.MatchList(title, matches, loc, now), nil
		})
	},
}

func init() {
	todayCmd.Flags().BoolVarP(&todayWatch, "watch", "w", false, "auto-refresh in place")
	rootCmd.AddCommand(todayCmd)
}
