package cmd

import (
	"fmt"

	"github.com/bdagnino/wcup/internal/ui"
	"github.com/spf13/cobra"
)

var standingsGroup string

var standingsCmd = &cobra.Command{
	Use:     "standings",
	Aliases: []string{"table", "groups"},
	Short:   "Group stage tables",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, _ := setup()
		groups, err := p.Standings(ctx)
		if err != nil {
			return err
		}
		if done, err := emitJSON(groups); done || err != nil {
			return err
		}
		fmt.Print(ui.Standings(groups, standingsGroup))
		return nil
	},
}

func init() {
	standingsCmd.Flags().StringVar(&standingsGroup, "group", "", "show only one group (A–L)")
	rootCmd.AddCommand(standingsCmd)
}
