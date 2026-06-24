package cmd

import (
	"fmt"
	"strings"

	"github.com/bdagnino/wc-cli/internal/provider"
	"github.com/bdagnino/wc-cli/internal/ui"
	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group <letter>",
	Short: "Standings for a single group (e.g. wcup group J)",
	Long: "Show the table for one group, A–L.\n\n" +
		"  wcup group J\n\n" +
		"Equivalent to `wcup standings --group J`; use `wcup standings` for all groups.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, p, _ := setup()
		letter := strings.ToUpper(strings.TrimSpace(args[0]))
		groups, err := p.Standings(ctx)
		if err != nil {
			return err
		}
		var found *provider.Group
		for i := range groups {
			if groups[i].Letter == letter {
				found = &groups[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("no group %q — World Cup groups run A–L", letter)
		}
		if done, err := emitJSON(*found); done || err != nil {
			return err
		}
		fmt.Print(ui.Standings(groups, letter))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(groupCmd)
}
