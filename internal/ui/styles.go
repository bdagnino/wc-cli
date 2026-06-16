package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Palette. Kept small and semantic so the whole CLI reads consistently.
var (
	cGreen  = lipgloss.Color("42")
	cYellow = lipgloss.Color("220")
	cGray   = lipgloss.Color("245")
	cFaint  = lipgloss.Color("240")
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
)

// SetColor enables or disables all styling. It honors --no-color and the
// NO_COLOR convention, auto-disables when output is not a terminal, and honors
// CLICOLOR_FORCE to keep color on when piped (e.g. capturing screenshots).
func SetColor(enabled bool) {
	if enabled {
		lipgloss.SetColorProfile(termenv.EnvColorProfile())
	} else {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
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
