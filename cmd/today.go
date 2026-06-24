package cmd

import (
	"github.com/spf13/cobra"
)

var todayWatch bool

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Today's matches with live scores",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDayView(todayWatch, "")
	},
}

func init() {
	todayCmd.Flags().BoolVarP(&todayWatch, "watch", "w", false, "auto-refresh in place")
	rootCmd.AddCommand(todayCmd)
}
