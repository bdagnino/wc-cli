package cmd

import (
	"time"

	"github.com/bdagnino/wcup/internal/ui"
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
			if done, err := emitJSON(matches); done || err != nil {
				return "", err
			}
			title := "Today · " + time.Now().In(loc).Format("Mon, Jan 2")
			return ui.MatchList(title, matches, loc), nil
		})
	},
}

func init() {
	todayCmd.Flags().BoolVarP(&todayWatch, "watch", "w", false, "auto-refresh in place")
	rootCmd.AddCommand(todayCmd)
}
