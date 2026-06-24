package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bdagnino/wc-cli/internal/espn"
	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagJSON    bool
	flagNoColor bool
	flagTZ      string
)

var rootCmd = &cobra.Command{
	Use:   "wcup",
	Short: "Follow the 2026 World Cup from your terminal",
	Long: "wcup shows World Cup 2026 scores, schedules, standings and teams\n" +
		"straight from the terminal. No account, no API key.",
	Example: "  wcup                         smart summary (live → today → next)\n" +
		"  wcup today                   today's matches, with live scores\n" +
		"  wcup day yesterday           another day (yesterday|tomorrow|YYYY-MM-DD)\n" +
		"  wcup live                    matches in progress right now\n" +
		"  wcup standings               all group tables\n" +
		"  wcup group J                 one group's table\n" +
		"  wcup match arg --last        a team's most recent finished match\n\n" +
		"Most list commands accept --team, --group, --date and --round filters.",
	SilenceUsage:  true,
	SilenceErrors: true,
	// Bare `wcup`: a smart summary — live now, else today, else next up.
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		now := time.Now().In(loc)
		today, err := p.Scoreboard(ctx, now)
		if err != nil {
			return err
		}
		live := filterState(today, provider.StateLive)
		if len(live) > 0 {
			fmt.Print(ui.MatchList("● Live now", live, loc, time.Time{}))
			return nil
		}
		if len(today) > 0 {
			fmt.Print(ui.MatchList("Today", today, loc, now))
			return nil
		}
		// Nothing today: show the next upcoming matches.
		sched, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		next := upcoming(sched, loc, 5)
		fmt.Print(ui.MatchList("Next up", next, loc, now))
		return nil
	},
}

// Execute is the program entry point. version is injected from main.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Faint.Render("error: ")+err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output raw JSON for scripting")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&flagTZ, "tz", "", "timezone for kickoff times, incl. JSON (e.g. Europe/Madrid); defaults to local")
}

// setup resolves shared dependencies once color/flags are known.
func setup() (context.Context, provider.Provider, *time.Location) {
	ui.SetColor(ui.ColorEnabled(flagNoColor))
	return context.Background(), espn.New(), location()
}

func location() *time.Location {
	if flagTZ == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(flagTZ)
	if err != nil {
		return time.Local
	}
	return loc
}

// emitJSON prints v as indented JSON and reports whether --json was set.
func emitJSON(v any) (bool, error) {
	if !flagJSON {
		return false, nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return true, enc.Encode(v)
}

// localize returns a copy of ms with kickoff times expressed in loc. ESPN
// timestamps are UTC, so without this the JSON "Kick" field always serializes
// with a "Z" suffix and a consumer has to convert it by hand. Expressing it in
// loc (the --tz zone, or the machine's local zone by default) makes the JSON
// carry the right offset (e.g. "+02:00") and match the human-readable output.
func localize(ms []provider.Match, loc *time.Location) []provider.Match {
	out := make([]provider.Match, len(ms))
	for i, m := range ms {
		m.Kick = m.Kick.In(loc)
		out[i] = m
	}
	return out
}

func filterState(ms []provider.Match, st provider.MatchState) []provider.Match {
	var out []provider.Match
	for _, m := range ms {
		if m.State == st {
			out = append(out, m)
		}
	}
	return out
}

func upcoming(ms []provider.Match, loc *time.Location, limit int) []provider.Match {
	now := time.Now()
	var out []provider.Match
	for _, m := range ms {
		if m.State == provider.StateScheduled && m.Kick.After(now) {
			out = append(out, m)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}
