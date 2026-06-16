package cmd

import (
	"time"

	"github.com/bdagnino/wcup/internal/provider"
	"github.com/bdagnino/wcup/internal/ui"
	"github.com/spf13/cobra"
)

var liveWatch bool

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "Matches in progress right now",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		return runWatch(liveWatch, func() (string, error) {
			matches, err := p.Scoreboard(ctx, time.Now().In(loc))
			if err != nil {
				return "", err
			}
			live := filterState(matches, provider.StateLive)
			if done, err := emitJSON(live); done || err != nil {
				return "", err
			}
			return ui.MatchList("● Live now", live, loc), nil
		})
	},
}

func init() {
	liveCmd.Flags().BoolVarP(&liveWatch, "watch", "w", false, "auto-refresh in place")
	rootCmd.AddCommand(liveCmd)
}
