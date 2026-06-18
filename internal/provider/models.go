package provider

import "time"

// MatchState is the high-level lifecycle of a match.
type MatchState string

const (
	StateScheduled MatchState = "scheduled"
	StateLive      MatchState = "live"
	StateFinished  MatchState = "finished"
)

// Team is a national team participating in the tournament.
type Team struct {
	// Abbr is the 3-letter code ESPN uses (e.g. "BRA"). It is the canonical
	// identifier callers pass to --team filters.
	Abbr string
	Name string
	// Group is the group letter (A–L) during the group stage; empty otherwise.
	Group string
}

// Match is a single fixture, scheduled, live, or finished.
type Match struct {
	ID    string
	Kick  time.Time
	State MatchState

	Home      Team
	Away      Team
	HomeScore int
	AwayScore int

	// Detail is the source's human label for the moment of the match,
	// e.g. "Scheduled", "45'+2'", "HT", "FT". Used as-is for display.
	Detail string
	// Clock is the running game clock for live matches, e.g. "67'".
	Clock string

	Venue string
	// Round is the stage label, e.g. "Group Stage", "Round of 16".
	Round string
	// Group is the group letter when applicable.
	Group string
}

// EventKind classifies a moment in a match timeline.
type EventKind string

const (
	EventGoal    EventKind = "goal"
	EventOwnGoal EventKind = "own_goal"
	EventPenalty EventKind = "penalty"
	EventYellow  EventKind = "yellow"
	EventRed     EventKind = "red"
	EventSub     EventKind = "sub"
	EventOther   EventKind = "other"
)

// MatchEvent is a single timeline entry (goal, card, substitution).
type MatchEvent struct {
	Clock   string // e.g. "67'"
	Kind    EventKind
	Players []string // scorer/assist, or card recipient, or [in, out] for subs
	Text    string   // source's full description
}

// MatchDetail enriches a Match with its timeline and game info.
type MatchDetail struct {
	Match
	Events     []MatchEvent
	Attendance int
}

// Scorer is one player's tally in the tournament's top-scorer (Golden Boot)
// race.
type Scorer struct {
	Rank     int
	Player   string
	TeamAbbr string
	Goals    int
}

// GroupStanding is one team's row within a group table.
type GroupStanding struct {
	Team         Team
	Rank         int
	Played       int
	Won          int
	Drawn        int
	Lost         int
	GoalsFor     int
	GoalsAgainst int
	GoalDiff     int
	Points       int
}

// Group is a labelled group with its ranked standings.
type Group struct {
	Letter    string
	Standings []GroupStanding
}
