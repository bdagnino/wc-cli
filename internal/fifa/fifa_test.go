package fifa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// calendarFixture has two matches sharing the ARG|GER pair (a group game and a
// later-round rematch) so the score tiebreak is exercised, plus an unfilled
// knockout slot that must be skipped.
const calendarFixture = `{"Results":[
  {"IdStage":"S1","IdMatch":"M_ARG_ALG","Home":{"IdCountry":"ARG"},"Away":{"IdCountry":"ALG"},"HomeTeamScore":3,"AwayTeamScore":0},
  {"IdStage":"S1","IdMatch":"M_ARG_GER_GRP","Home":{"IdCountry":"ARG"},"Away":{"IdCountry":"GER"},"HomeTeamScore":1,"AwayTeamScore":1},
  {"IdStage":"S5","IdMatch":"M_GER_ARG_FINAL","Home":{"IdCountry":"GER"},"Away":{"IdCountry":"ARG"},"HomeTeamScore":2,"AwayTeamScore":0},
  {"IdStage":"S2","IdMatch":"M_TBD","Home":{"IdCountry":""},"Away":{"IdCountry":""},"HomeTeamScore":null,"AwayTeamScore":null}
]}`

func videosFor(matchID string) string {
	switch matchID {
	case "M_ARG_ALG":
		return `{"vodVideosBaseCarousel":{"items":[
		  {"videoSubcategory":"Highlights","readMorePageUrl":"/en/watch/ARGALG"}]}}`
	case "M_GER_ARG_FINAL":
		return `{"vodVideosBaseCarousel":{"items":[
		  {"videoSubcategory":"Highlights","readMorePageUrl":"/en/watch/FINAL"}]}}`
	default:
		// No highlights published yet for this match.
		return `{"vodVideosBaseCarousel":{"items":[
		  {"videoSubcategory":"Match Review","readMorePageUrl":"/en/watch/REVIEW"}]}}`
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/calendar/matches", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(calendarFixture))
	})
	mux.HandleFunc("/sections/matchdetails/videos", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(videosFor(r.URL.Query().Get("matchId"))))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), apiBase: srv.URL, cxmBase: srv.URL}
}

func TestHighlightsResolvesExactLink(t *testing.T) {
	c := newTestClient(t)
	got, err := c.Highlights(context.Background(), "ARG", "ALG", 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := siteBase + "/en/watch/ARGALG"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestHighlightsCodeOrderIndependent(t *testing.T) {
	c := newTestClient(t)
	// Querying away-first must resolve the same match.
	got, err := c.Highlights(context.Background(), "ALG", "ARG", 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if want := siteBase + "/en/watch/ARGALG"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestHighlightsDisambiguatesByScore(t *testing.T) {
	c := newTestClient(t)
	// ARG and GER meet twice; the 2-0 final must resolve to the final reel,
	// not the 1-1 group game.
	got, err := c.Highlights(context.Background(), "GER", "ARG", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := siteBase + "/en/watch/FINAL"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestHighlightsEmptyWhenNoneOrUnknown(t *testing.T) {
	c := newTestClient(t)
	// Pair with no Highlights item.
	if got, err := c.Highlights(context.Background(), "ARG", "GER", 1, 1); err != nil || got != "" {
		t.Fatalf("got (%q, %v), want empty", got, err)
	}
	// Unknown pair entirely.
	if got, err := c.Highlights(context.Background(), "BRA", "FRA", 0, 0); err != nil || got != "" {
		t.Fatalf("got (%q, %v), want empty", got, err)
	}
}
