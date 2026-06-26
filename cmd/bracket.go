package cmd

import (
	"fmt"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var bracketCmd = &cobra.Command{
	Use:     "bracket [team]",
	Aliases: []string{"ko"},
	Short:   "Knockout bracket — the whole tree, or one team's path",
	Long: "Draw the knockout stage as a bracket.\n\n" +
		"  wcup bracket            the full tree, Round of 32 to the final\n" +
		"  wcup bracket argentina  one team's road to the final",
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

		b, ok := ui.BuildBracket(ko)
		if !ok {
			// Fixtures exist but the tree isn't fully formed yet — fall back to
			// a plain per-round listing rather than a broken diagram.
			return renderRoundList(ko, loc)
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
	rootCmd.AddCommand(bracketCmd)
}
