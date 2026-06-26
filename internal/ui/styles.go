package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Palette. Kept small and semantic so the whole CLI reads consistently.
// The grays are deliberately on the light side of the 256-color ramp: 240
// (#585858) is too dark to read on most terminal backgrounds, so secondary
// text uses 250 and the faintest tier uses 245.
var (
	cGreen  = lipgloss.Color("42")
	cYellow = lipgloss.Color("220")
	cGray   = lipgloss.Color("250")
	cFaint  = lipgloss.Color("245")
	cWhite  = lipgloss.Color("255")
	cAccent = lipgloss.Color("39")
)

var (
	Title    = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	Header   = lipgloss.NewStyle().Bold(true).Foreground(cWhite)
	Faint    = lipgloss.NewStyle().Foreground(cFaint)
	Muted    = lipgloss.NewStyle().Foreground(cGray)
	Score    = lipgloss.NewStyle().Bold(true).Foreground(cWhite)
	Live     = lipgloss.NewStyle().Bold(true).Foreground(cGreen)
	Upcoming = lipgloss.NewStyle().Foreground(cYellow)
	Winner   = lipgloss.NewStyle().Bold(true).Foreground(cGreen)
	// Pencil marks a team penciled into the bracket from current standings —
	// tentative, so italic yellow rather than the confirmed bold white.
	Pencil = lipgloss.NewStyle().Italic(true).Foreground(cYellow)
)

// SetColor enables or disables all styling. It honors --no-color and the
// NO_COLOR convention, auto-disables when output is not a terminal, and honors
// CLICOLOR_FORCE to keep color on when piped (e.g. capturing screenshots).
func SetColor(enabled bool) {
	if !enabled {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}
	// When color is explicitly forced on for a non-TTY (e.g. piping into freeze
	// to capture screenshots), EnvColorProfile downgrades to 16-color ANSI,
	// which mangles the 256-color palette. Force full truecolor so piped output
	// keeps the real colors.
	if _, forced := os.LookupEnv("CLICOLOR_FORCE"); forced {
		lipgloss.SetColorProfile(termenv.TrueColor)
		return
	}
	lipgloss.SetColorProfile(termenv.EnvColorProfile())
}

// ColorEnabled resolves whether color should be on given the flag.
func ColorEnabled(noColorFlag bool) bool {
	if noColorFlag {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return true
}
