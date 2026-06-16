package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bdagnino/wcup/internal/ui"
)

const watchInterval = 30 * time.Second

// runWatch repeatedly renders frame in place until interrupted. When watch is
// false it renders a single frame and returns. frame returns the full screen
// content to display.
func runWatch(watch bool, frame func() (string, error)) error {
	if !watch {
		s, err := frame()
		if err != nil {
			return err
		}
		fmt.Print(s)
		return nil
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	for {
		s, err := frame()
		if err != nil {
			return err
		}
		clearScreen()
		fmt.Print(s)
		fmt.Print(ui.Faint.Render(fmt.Sprintf("\nrefreshing every %ds · press Ctrl-C to exit\n", int(watchInterval.Seconds()))))

		select {
		case <-sig:
			fmt.Println()
			return nil
		case <-ticker.C:
		}
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
