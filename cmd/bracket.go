package cmd

import (
	"fmt"

	"github.com/bdagnino/wcup/internal/provider"
	"github.com/bdagnino/wcup/internal/ui"
	"github.com/spf13/cobra"
)

var bracketCmd = &cobra.Command{
	Use:     "bracket",
	Aliases: []string{"ko"},
	Short:   "Knockout bracket (Round of 32 onward)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}

		// Knockout = everything that is not the group stage.
		var ko []provider.Match
		for _, m := range all {
			if !roundMatches(m.Round, "group") {
				ko = append(ko, m)
			}
		}
		if done, err := emitJSON(ko); done || err != nil {
			return err
		}
		if len(ko) == 0 {
			fmt.Println(ui.Muted.Render("The knockout stage hasn't started yet — check back after the group stage."))
			return nil
		}

		// Render grouped by round, in tournament order.
		order := []struct{ token, title string }{
			{"r32", "Round of 32"},
			{"r16", "Round of 16"},
			{"qf", "Quarterfinals"},
			{"sf", "Semifinals"},
			{"final", "Final"},
		}
		for _, r := range order {
			var rms []provider.Match
			for _, m := range ko {
				if roundMatches(m.Round, r.token) {
					rms = append(rms, m)
				}
			}
			if len(rms) > 0 {
				fmt.Print(ui.MatchList(r.title, rms, loc))
				fmt.Println()
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bracketCmd)
}
