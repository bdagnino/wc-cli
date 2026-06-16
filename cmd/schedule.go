package cmd

import (
	"fmt"

	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var scheduleOpts filterOpts

var scheduleCmd = &cobra.Command{
	Use:     "schedule",
	Aliases: []string{"fixtures"},
	Short:   "Upcoming match schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, loc := setup()
		all, err := p.Schedule(ctx)
		if err != nil {
			return err
		}
		matches := scheduleOpts.apply(all, loc)
		if done, err := emitJSON(matches); done || err != nil {
			return err
		}
		fmt.Print(ui.MatchListByDay("Schedule", matches, loc))
		return nil
	},
}

func init() {
	addFilterFlags(scheduleCmd, &scheduleOpts)
	rootCmd.AddCommand(scheduleCmd)
}

// addFilterFlags wires the shared filter flags onto a command.
func addFilterFlags(c *cobra.Command, o *filterOpts) {
	c.Flags().StringVar(&o.team, "team", "", "filter by team (name or code, e.g. BRA)")
	c.Flags().StringVar(&o.group, "group", "", "filter by group letter (A–L)")
	c.Flags().StringVar(&o.date, "date", "", "filter by date (today, tomorrow, YYYY-MM-DD)")
	c.Flags().StringVar(&o.round, "round", "", "filter by round (group, r32, r16, qf, sf, final)")
	c.Flags().IntVarP(&o.limit, "limit", "n", 0, "limit number of matches shown")
}
