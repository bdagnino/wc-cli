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
	// Scorers returns the tournament's top scorers, most goals first, limited
	// to the top n (n <= 0 means all available).
	Scorers(ctx context.Context, n int) ([]Scorer, error)
	// Detail returns a single match's timeline and game info by its ID.
	Detail(ctx context.Context, id string) (MatchDetail, error)
	// BracketOrder returns the canonical bracket position (the source's match
	// number) for each given match id. The knockout tree is ordered by this,
	// not by event id. Ids whose number can't be resolved are omitted.
	BracketOrder(ctx context.Context, ids []string) (map[string]int, error)
}
