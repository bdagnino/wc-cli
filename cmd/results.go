package cmd

import (
	"fmt"

	"github.com/bdagnino/wcup/internal/provider"
	"github.com/bdagnino/wcup/internal/ui"
	"github.com/spf13/cobra"
)

var resultsOpts filterOpts

var resultsCmd = &cobra.Command{
	Use:     "results",
	Aliases: []string{"scores"},
	Short:   "Recently finished matches",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		finished := filterState(all, provider.StateFinished)
		// Most recent first.
		reverse(finished)
		matches := resultsOpts.apply(finished, loc)
		if done, err := emitJSON(matches); done || err != nil {
			return err
		}
		fmt.Print(ui.MatchListByDay("Results", matches, loc))
		return nil
	},
}

func init() {
	addFilterFlags(resultsCmd, &resultsOpts)
	rootCmd.AddCommand(resultsCmd)
}

func reverse(ms []provider.Match) {
	for i, j := 0, len(ms)-1; i < j; i, j = i+1, j-1 {
		ms[i], ms[j] = ms[j], ms[i]
	}
}
