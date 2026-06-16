package cmd

import (
	"fmt"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	nextTeam  string
	nextLimit int
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "The next upcoming match (optionally for a team)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		f := filterOpts{team: nextTeam, limit: nextLimit}
		future := upcoming(all, loc, 0) // all future scheduled matches
		matches := f.apply(future, loc)
		if done, err := emitJSON(matches); done || err != nil {
			return err
		}
		title := "Next up"
		if nextTeam != "" {
			title = "Next up · " + nextTeam
		}
		fmt.Print(ui.MatchList(title, matches, loc))
		return nil
	},
}

func init() {
	nextCmd.Flags().StringVar(&nextTeam, "team", "", "team to look up (name or code, e.g. BRA)")
	nextCmd.Flags().IntVarP(&nextLimit, "limit", "n", 1, "how many upcoming matches to show")
	rootCmd.AddCommand(nextCmd)
}
