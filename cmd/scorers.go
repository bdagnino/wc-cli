package cmd

import (
	"fmt"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var scorersLimit int

var scorersCmd = &cobra.Command{
	Use:     "scorers",
	Aliases: []string{"goals", "topscorers", "goldenboot"},
	Short:   "Top scorers — the Golden Boot race",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, _ := setup()
		scorers, err := p.Scorers(ctx, scorersLimit)
		if err != nil {
			return err
		}
		if done, err := emitJSON(scorers); done || err != nil {
			return err
		}
		fmt.Print(ui.Scorers(scorers))
		return nil
	},
}

func init() {
	scorersCmd.Flags().IntVarP(&scorersLimit, "limit", "n", 10, "how many scorers to show")
	rootCmd.AddCommand(scorersCmd)
}
