package cmd

import (
	"fmt"
	"strings"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	teamsGroup string
	teamsFlat  bool
)

var teamsCmd = &cobra.Command{
	Use:   "teams [query]",
	Short: "List teams (or search for one)",
	Long:  "List all participating teams grouped by group. Pass a query to fuzzy-search,\ne.g. `wcup teams cong` finds COD Congo DR.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, _ := setup()
		teams, err := p.Teams(ctx)
		if err != nil {
			return err
		}

		// Search mode: a positional query.
		if len(args) > 0 {
			matches := provider.FindTeams(teams, args[0])
			results := make([]provider.Team, 0, len(matches))
			for _, m := range matches {
				results = append(results, m.Team)
			}
			if done, err := emitJSON(results); done || err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Println(ui.Muted.Render("No team matches \"" + args[0] + "\". Run `wcup teams` to see them all."))
				return nil
			}
			fmt.Print(ui.Teams(results, true))
			return nil
		}

		// Group filter.
		if teamsGroup != "" {
			var filtered []provider.Team
			for _, t := range teams {
				if strings.EqualFold(t.Group, teamsGroup) {
					filtered = append(filtered, t)
				}
			}
			teams = filtered
		}

		if done, err := emitJSON(teams); done || err != nil {
			return err
		}
		fmt.Print(ui.Teams(teams, teamsFlat))
		return nil
	},
}

func init() {
	teamsCmd.Flags().StringVar(&teamsGroup, "group", "", "show only one group (A–L)")
	teamsCmd.Flags().BoolVar(&teamsFlat, "flat", false, "flat alphabetical list instead of grouped")
	rootCmd.AddCommand(teamsCmd)
}
