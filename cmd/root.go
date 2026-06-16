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
	SilenceUsage:  true,
	SilenceErrors: true,
	// Bare `wcup`: a smart summary — live now, else today, else next up.
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		today, err := p.Scoreboard(ctx, time.Now().In(loc))
		if err != nil {
			return err
		}
		live := filterState(today, provider.StateLive)
		if len(live) > 0 {
			fmt.Print(ui.MatchList("● Live now", live, loc))
			return nil
		}
		if len(today) > 0 {
			fmt.Print(ui.MatchList("Today", today, loc))
			return nil
		}
		// Nothing today: show the next upcoming matches.
		sched, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		next := upcoming(sched, loc, 5)
		fmt.Print(ui.MatchList("Next up", next, loc))
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
	rootCmd.PersistentFlags().StringVar(&flagTZ, "tz", "", "timezone for kickoff times (e.g. Europe/Madrid); defaults to local")
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
