package provider

import (
	"context"
	"time"
)

// Provider is the data source for tournament information. Commands depend only
// on this interface, never on a concrete source, so a new backend can be added
// without touching command code.
type Provider interface {
	// Scoreboard returns the matches on a given day (caller's timezone is
	// irrelevant; pass the calendar day you want). A zero day means "today".
	Scoreboard(ctx context.Context, day time.Time) ([]Match, error)
	// Schedule returns matches across the whole tournament window.
	Schedule(ctx context.Context) ([]Match, error)
	// Teams returns all participating teams.
	Teams(ctx context.Context) ([]Team, error)
	// Standings returns the group tables.
	Standings(ctx context.Context) ([]Group, error)
	// Detail returns a single match's timeline and game info by its ID.
	Detail(ctx context.Context, id string) (MatchDetail, error)
}
