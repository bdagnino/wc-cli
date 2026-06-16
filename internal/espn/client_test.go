package espn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bdagnino/wcup/internal/provider"
)

// newTestClient spins an in-process server that serves canned ESPN payloads,
// so the parsing logic is exercised without touching the network.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	serve := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}
	}
	mux.HandleFunc("/site/v2/sports/soccer/fifa.world/scoreboard", serve(scoreboardFixture))
	mux.HandleFunc("/v2/sports/soccer/fifa.world/standings", serve(standingsFixture))
	mux.HandleFunc("/site/v2/sports/soccer/fifa.world/teams", serve(teamsFixture))
	mux.HandleFunc("/site/v2/sports/soccer/fifa.world/summary", serve(summaryFixture))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), base: srv.URL}
}

func TestScoreboardParsing(t *testing.T) {
	c := newTestClient(t)
	ms, err := c.Scoreboard(context.Background(), time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("got %d matches, want 2", len(ms))
	}
	// Sorted by kickoff: finished BRA–ARG first, scheduled FRA–ESP second.
	first := ms[0]
	if first.State != provider.StateFinished {
		t.Errorf("first state = %q, want finished", first.State)
	}
	if first.Home.Abbr != "BRA" || first.HomeScore != 2 || first.AwayScore != 1 {
		t.Errorf("first scoreline wrong: %+v", first)
	}
	if first.Venue != "Stadium X" || first.Round != "Group Stage" {
		t.Errorf("first venue/round wrong: %q / %q", first.Venue, first.Round)
	}
	// Group enrichment from standings: BRA is in Group A; FRA is absent.
	if first.Group != "A" {
		t.Errorf("first group = %q, want A", first.Group)
	}
	if want := time.Date(2026, 6, 16, 19, 0, 0, 0, time.UTC); !first.Kick.Equal(want) {
		t.Errorf("first kick = %v, want %v (seconds-less parse)", first.Kick, want)
	}
	if ms[1].State != provider.StateScheduled || ms[1].Group != "" {
		t.Errorf("second match wrong: state=%q group=%q", ms[1].State, ms[1].Group)
	}
}

func TestStandingsParsing(t *testing.T) {
	c := newTestClient(t)
	groups, err := c.Standings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Letter != "A" {
		t.Fatalf("groups = %+v, want one group A", groups)
	}
	top := groups[0].Standings[0] // sorted by rank
	if top.Team.Abbr != "BRA" || top.Rank != 1 || top.Points != 3 || top.GoalDiff != 1 || top.Won != 1 {
		t.Errorf("top row wrong: %+v", top)
	}
}

func TestTeamsParsing(t *testing.T) {
	c := newTestClient(t)
	teams, err := c.Teams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 || teams[0].Name != "Brazil" { // sorted alphabetically
		t.Fatalf("teams = %+v", teams)
	}
	if teams[0].Group != "A" {
		t.Errorf("Brazil group = %q, want A", teams[0].Group)
	}
}

func TestDetailParsing(t *testing.T) {
	c := newTestClient(t)
	d, err := c.Detail(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Attendance != 80000 {
		t.Errorf("attendance = %d, want 80000", d.Attendance)
	}
	if len(d.Events) != 2 { // kickoff is filtered out
		t.Fatalf("got %d events, want 2 (kickoff filtered)", len(d.Events))
	}
	g := d.Events[0]
	if g.Kind != provider.EventGoal || len(g.Players) != 2 || g.Players[0] != "Scorer One" {
		t.Errorf("goal event wrong: %+v", g)
	}
	if d.Events[1].Kind != provider.EventYellow {
		t.Errorf("second event kind = %q, want yellow", d.Events[1].Kind)
	}
}

func TestEventKind(t *testing.T) {
	cases := map[string]provider.EventKind{
		"Goal":             provider.EventGoal,
		"Own Goal":         provider.EventOwnGoal,
		"Penalty - Scored": provider.EventPenalty,
		"Yellow Card":      provider.EventYellow,
		"Red Card":         provider.EventRed,
		"Substitution":     provider.EventSub,
		"Kickoff":          provider.EventOther,
	}
	for in, want := range cases {
		if got := eventKind(in); got != want {
			t.Errorf("eventKind(%q) = %q, want %q", in, got, want)
		}
	}
	// Own goal must win over the bare "goal" substring.
	if eventKind("Own Goal by ...") != provider.EventOwnGoal {
		t.Error("own goal misclassified")
	}
}

const scoreboardFixture = `{
  "leagues":[{"season":{"type":{"name":"Group Stage"}}}],
  "events":[
    {"id":"1","date":"2026-06-16T19:00Z","name":"Argentina at Brazil",
     "competitions":[{"venue":{"fullName":"Stadium X"},
       "status":{"type":{"state":"post","shortDetail":"FT"}},
       "competitors":[
         {"homeAway":"home","score":"2","team":{"abbreviation":"BRA","displayName":"Brazil"}},
         {"homeAway":"away","score":"1","team":{"abbreviation":"ARG","displayName":"Argentina"}}
       ]}]},
    {"id":"2","date":"2026-06-16T22:00Z","name":"Spain at France",
     "competitions":[{"venue":{"fullName":"Stadium Y"},
       "status":{"type":{"state":"pre","shortDetail":"Scheduled"}},
       "competitors":[
         {"homeAway":"home","score":"0","team":{"abbreviation":"FRA","displayName":"France"}},
         {"homeAway":"away","score":"0","team":{"abbreviation":"ESP","displayName":"Spain"}}
       ]}]}
  ]
}`

const standingsFixture = `{
  "children":[{"name":"Group A","standings":{"entries":[
    {"team":{"abbreviation":"BRA","displayName":"Brazil"},"stats":[
      {"name":"rank","value":1},{"name":"gamesPlayed","value":1},{"name":"wins","value":1},
      {"name":"ties","value":0},{"name":"losses","value":0},{"name":"pointsFor","value":2},
      {"name":"pointsAgainst","value":1},{"name":"pointDifferential","value":1},{"name":"points","value":3}]},
    {"team":{"abbreviation":"ARG","displayName":"Argentina"},"stats":[
      {"name":"rank","value":2},{"name":"points","value":0}]}
  ]}}]
}`

const teamsFixture = `{
  "sports":[{"leagues":[{"teams":[
    {"team":{"abbreviation":"BRA","displayName":"Brazil"}},
    {"team":{"abbreviation":"FRA","displayName":"France"}}
  ]}]}]
}`

const summaryFixture = `{
  "gameInfo":{"attendance":80000},
  "keyEvents":[
    {"clock":{"displayValue":"23'"},"type":{"text":"Goal"},"text":"Goal!",
     "participants":[{"athlete":{"displayName":"Scorer One"}},{"athlete":{"displayName":"Assister Two"}}]},
    {"clock":{"displayValue":"45'"},"type":{"text":"Yellow Card"},"text":"booked",
     "participants":[{"athlete":{"displayName":"Carded Three"}}]},
    {"clock":{"displayValue":"1'"},"type":{"text":"Kickoff"},"text":"start","participants":[]}
  ]
}`
