package cmd

import (
	"fmt"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var bracketLive bool

var bracketCmd = &cobra.Command{
	Use:     "bracket [team]",
	Aliases: []string{"ko"},
	Short:   "Knockout bracket — the whole tree, or one team's path",
	Long: "Draw the knockout stage as a bracket.\n\n" +
		"  wcup bracket             the full tree, Round of 32 to the final\n" +
		"  wcup bracket argentina   one team's road to the final\n" +
		"  wcup bracket --live      pencil in teams from the current standings",
	Args: cobra.MaximumNArgs(1),
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
		if done, err := emitJSON(localize(ko, loc)); done || err != nil {
			return err
		}
		if len(ko) == 0 {
			fmt.Println(ui.Muted.Render("The knockout stage hasn't started yet — check back after the group stage."))
			return nil
		}

		// Order the tree by the source's canonical match numbers. Best-effort:
		// if the lookup fails, BuildBracket falls back to event-id order.
		ids := make([]string, len(ko))
		for i := range ko {
			ids[i] = ko[i].ID
		}
		if nums, err := p.BracketOrder(ctx, ids); err == nil {
			for i := range ko {
				ko[i].MatchNumber = nums[ko[i].ID]
			}
		}

		b, ok := ui.BuildBracket(ko)
		if !ok {
			// Fixtures exist but the tree isn't fully formed yet — fall back to
			// a plain per-round listing rather than a broken diagram.
			return renderRoundList(ko, loc)
		}

		// --live pencils empty group slots in from the current standings: if the
		// group tables held as they are now, these are the teams that would land
		// in each Round of 32 slot.
		projected := 0
		if bracketLive {
			groups, err := p.Standings(ctx)
			if err != nil {
				return err
			}
			projected = b.Project(groups)
		}

		if len(args) == 1 {
			path, found := b.Path(args[0], loc)
			if !found {
				return fmt.Errorf("no knockout team matching %q (try a code like ARG, or see `wcup teams`)", args[0])
			}
			fmt.Print(path)
			return nil
		}

		fmt.Print(b.Render(loc))
		if bracketLive {
			fmt.Println()
			if projected > 0 {
				fmt.Println(ui.Pencil.Render("penciled in") +
					ui.Muted.Render(" — teams the current group standings would send here, if they held."))
			} else {
				fmt.Println(ui.Muted.Render("Nothing to pencil in yet — the group standings haven't taken shape."))
			}
		}
		return nil
	},
}

// renderRoundList is the fallback view: knockout matches grouped by round.
func renderRoundList(ko []provider.Match, loc *time.Location) error {
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
			fmt.Print(ui.MatchList(r.title, rms, loc, time.Time{}))
			fmt.Println()
		}
	}
	return nil
}

func init() {
	bracketCmd.Flags().BoolVar(&bracketLive, "live", false,
		"pencil empty slots in with the teams the current group standings would send there")
	rootCmd.AddCommand(bracketCmd)
}
