package cmd

import (
	"fmt"

	"github.com/bdagnino/wcup/internal/provider"
	"github.com/bdagnino/wcup/internal/ui"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team <name>",
	Short: "A team's fixtures, results and group",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		query := joinArgs(args)

		teams, err := p.Teams(ctx)
		if err != nil {
			return err
		}
		team, ok := provider.FindTeam(teams, query)
		if !ok {
			fmt.Println(ui.Muted.Render("No team matches \"" + query + "\". Run `wcup teams` to see them all."))
			return nil
		}

		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		var theirs []provider.Match
		for _, m := range all {
			if m.Home.Abbr == team.Abbr || m.Away.Abbr == team.Abbr {
				theirs = append(theirs, m)
			}
		}

		if done, err := emitJSON(map[string]any{"team": team, "matches": theirs}); done || err != nil {
			return err
		}

		header := ui.Flag(team.Abbr) + " " + ui.Title.Render(team.Name)
		if team.Group != "" {
			header += ui.Faint.Render("  ·  Group " + team.Group)
		}
		fmt.Println(header)
		fmt.Print(ui.MatchListByDay("", theirs, loc))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(teamCmd)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
