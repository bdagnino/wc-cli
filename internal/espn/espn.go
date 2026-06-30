// Package espn implements provider.Provider against ESPN's public, no-auth
// soccer API for the FIFA World Cup.
package espn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
)

const defaultBase = "https://site.api.espn.com/apis"

// defaultCoreBase is the "core" hypermedia API, used for season-wide leaders
// (top scorers) which the site API doesn't expose.
const defaultCoreBase = "https://sports.core.api.espn.com/v2/sports/soccer/leagues/fifa.world"

const (
	scorersSeason = "2026"
	// scorersType 0 is the whole-tournament aggregate, so the Golden Boot race
	// keeps accumulating across the group stage and knockouts (a per-stage type
	// would reset).
	scorersType = "0"
)

// Client is an ESPN-backed provider.
type Client struct {
	HTTP *http.Client
	// base is the site API root; coreBase is the core API root. Both are
	// overridable in tests.
	base     string
	coreBase string
}

// New returns a Client with sensible network defaults.
func New() *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: 15 * time.Second},
		base:     defaultBase,
		coreBase: defaultCoreBase,
	}
}

func (c *Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("reaching ESPN: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ESPN returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding ESPN response: %w", err)
	}
	return nil
}

// Scoreboard returns the matches for a single calendar day. A zero day means
// the whole tournament's default scoreboard (effectively today).
func (c *Client) Scoreboard(ctx context.Context, day time.Time) ([]provider.Match, error) {
	url := c.base + "/site/v2/sports/soccer/fifa.world/scoreboard"
	if !day.IsZero() {
		url += "?dates=" + day.Format("20060102")
	}
	var raw scoreboardResp
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	groups := c.groupByTeam(ctx) // best-effort enrichment
	matches := make([]provider.Match, 0, len(raw.Events))
	for _, e := range raw.Events {
		if m, ok := toMatch(e, raw.round(), groups); ok {
			clampGroup(&m)
			matches = append(matches, m)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Kick.Before(matches[j].Kick) })
	return matches, nil
}

// Schedule returns every tournament match in a single ranged request. Because
// a ranged response carries no per-day stage label, the round of each match is
// resolved from the league calendar's date windows.
func (c *Client) Schedule(ctx context.Context) ([]provider.Match, error) {
	groups := c.groupByTeam(ctx)
	resolver := c.roundResolver(ctx)

	// Tournament window: 2026-06-11 .. 2026-07-19. limit lifts the default 100.
	url := c.base + "/site/v2/sports/soccer/fifa.world/scoreboard?dates=20260611-20260719&limit=300"
	var raw scoreboardResp
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}

	all := make([]provider.Match, 0, len(raw.Events))
	for _, e := range raw.Events {
		m, ok := toMatch(e, "", groups)
		if !ok {
			continue
		}
		if r := resolver(m.Kick); r != "" {
			m.Round = r
		}
		clampGroup(&m)
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Kick.Before(all[j].Kick) })
	return all, nil
}

// roundResolver fetches the league calendar once and returns a function mapping
// a kickoff time to its stage label. On any failure it returns a resolver that
// yields "", so callers degrade gracefully rather than error.
func (c *Client) roundResolver(ctx context.Context) func(time.Time) string {
	var raw scoreboardResp
	if err := c.get(ctx, c.base+"/site/v2/sports/soccer/fifa.world/scoreboard", &raw); err != nil {
		return func(time.Time) string { return "" }
	}
	type window struct {
		label      string
		start, end time.Time
	}
	var windows []window
	for _, e := range raw.calendarEntries() {
		s, err1 := parseTime(e.StartDate)
		end, err2 := parseTime(e.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		windows = append(windows, window{label: e.Label, start: s, end: end})
	}
	return func(t time.Time) string {
		for _, w := range windows {
			if !t.Before(w.start) && t.Before(w.end) {
				return w.label
			}
		}
		return ""
	}
}

// Teams returns all participating teams, enriched with group letters from the
// standings endpoint when available.
func (c *Client) Teams(ctx context.Context) ([]provider.Team, error) {
	var raw teamsResp
	url := c.base + "/site/v2/sports/soccer/fifa.world/teams"
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	groups := c.groupByTeam(ctx)
	var teams []provider.Team
	for _, s := range raw.Sports {
		for _, l := range s.Leagues {
			for _, t := range l.Teams {
				teams = append(teams, provider.Team{
					Abbr:  t.Team.Abbreviation,
					Name:  t.Team.DisplayName,
					Group: groups[t.Team.Abbreviation],
				})
			}
		}
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })
	return teams, nil
}

// Standings returns the group tables (A–L).
func (c *Client) Standings(ctx context.Context) ([]provider.Group, error) {
	var raw standingsResp
	url := c.base + "/v2/sports/soccer/fifa.world/standings"
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	var groups []provider.Group
	for _, ch := range raw.Children {
		letter := groupLetter(ch.Name)
		g := provider.Group{Letter: letter}
		for _, en := range ch.Standings.Entries {
			st := provider.GroupStanding{
				Team: provider.Team{Abbr: en.Team.Abbreviation, Name: en.Team.DisplayName, Group: letter},
			}
			for _, s := range en.Stats {
				v := int(s.Value)
				switch s.Name {
				case "rank":
					st.Rank = v
				case "gamesPlayed":
					st.Played = v
				case "wins":
					st.Won = v
				case "ties":
					st.Drawn = v
				case "losses":
					st.Lost = v
				case "pointsFor":
					st.GoalsFor = v
				case "pointsAgainst":
					st.GoalsAgainst = v
				case "pointDifferential":
					st.GoalDiff = v
				case "points":
					st.Points = v
				}
			}
			g.Standings = append(g.Standings, st)
		}
		sort.SliceStable(g.Standings, func(i, j int) bool { return g.Standings[i].Rank < g.Standings[j].Rank })
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Letter < groups[j].Letter })
	return groups, nil
}

// Detail returns a match's timeline (goals, cards, substitutions) and game
// info. The caller already holds the Match metadata; this fills in Events and
// Attendance, leaving the embedded Match zero for the caller to populate.
func (c *Client) Detail(ctx context.Context, id string) (provider.MatchDetail, error) {
	url := c.base + "/site/v2/sports/soccer/fifa.world/summary?event=" + id
	var raw summaryResp
	if err := c.get(ctx, url, &raw); err != nil {
		return provider.MatchDetail{}, err
	}
	d := provider.MatchDetail{Attendance: raw.GameInfo.Attendance}
	for _, e := range raw.KeyEvents {
		kind := eventKind(e.Type.Text)
		if kind == provider.EventOther {
			continue // skip kickoff, delays, period markers
		}
		var players []string
		for _, p := range e.Participants {
			if n := p.Athlete.DisplayName; n != "" {
				players = append(players, n)
			}
		}
		d.Events = append(d.Events, provider.MatchEvent{
			Clock:   e.Clock.DisplayValue,
			Kind:    kind,
			Players: players,
			Text:    e.Text,
		})
	}
	return d, nil
}

// Scorers returns the tournament's top scorers (most goals first), limited to
// the top n. The leaders feed references each athlete by link, so names are
// resolved with one concurrent request per scorer.
func (c *Client) Scorers(ctx context.Context, n int) ([]provider.Scorer, error) {
	url := fmt.Sprintf("%s/seasons/%s/types/%s/leaders?lang=en", c.coreBase, scorersSeason, scorersType)
	var raw leadersResp
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	leaders := raw.goalsLeaders()
	if n > 0 && len(leaders) > n {
		leaders = leaders[:n]
	}

	out := make([]provider.Scorer, len(leaders))
	var wg sync.WaitGroup
	rank := 1
	for i, l := range leaders {
		// Standard competition ranking: equal goals share a rank.
		if i > 0 && l.Value < leaders[i-1].Value {
			rank = i + 1
		}
		out[i] = provider.Scorer{Rank: rank, Goals: int(l.Value)}
		wg.Add(1)
		go func(i int, ref string) {
			defer wg.Done()
			out[i].Player, out[i].TeamAbbr = c.athlete(ctx, ref)
		}(i, l.Athlete.Ref)
	}
	wg.Wait()
	return out, nil
}

// BracketOrder resolves each match id to its canonical bracket position. ESPN
// only exposes that number ("matchNumber") on the core-API event document, not
// the scoreboard, so it's fetched with one concurrent request per id. Ids that
// fail to resolve are left out, so a flaky row can't break the ordering.
func (c *Client) BracketOrder(ctx context.Context, ids []string) (map[string]int, error) {
	nums := make([]int, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			nums[i] = c.matchNumber(ctx, id)
		}(i, id)
	}
	wg.Wait()
	out := make(map[string]int, len(ids))
	for i, id := range ids {
		if nums[i] > 0 {
			out[id] = nums[i]
		}
	}
	return out, nil
}

// matchNumber reads a single event's bracket position from the core API, or 0
// if it's absent or the request fails.
func (c *Client) matchNumber(ctx context.Context, id string) int {
	var raw struct {
		Competitions []struct {
			MatchNumber int `json:"matchNumber"`
		} `json:"competitions"`
	}
	if err := c.get(ctx, fmt.Sprintf("%s/events/%s?lang=en", c.coreBase, id), &raw); err != nil {
		return 0
	}
	if len(raw.Competitions) == 0 {
		return 0
	}
	return raw.Competitions[0].MatchNumber
}

// athlete resolves an athlete link to a display name and 3-letter country code.
// Failures degrade to empty strings so one bad row doesn't sink the table.
func (c *Client) athlete(ctx context.Context, ref string) (name, code string) {
	if ref == "" {
		return "", ""
	}
	var a athleteResp
	if err := c.get(ctx, httpsURL(ref), &a); err != nil {
		return "", ""
	}
	return a.DisplayName, countryFromFlag(a.Flag.Href)
}

// countryFromFlag pulls "ARG" out of a flag URL like ".../countries/500/arg.png".
func countryFromFlag(href string) string {
	if href == "" {
		return ""
	}
	base := href[strings.LastIndex(href, "/")+1:]
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	return strings.ToUpper(base)
}

// httpsURL upgrades an http link to https. ESPN's core API returns http
// $ref links; we keep all traffic on https.
func httpsURL(u string) string {
	if strings.HasPrefix(u, "http://") {
		return "https://" + u[len("http://"):]
	}
	return u
}

func eventKind(text string) provider.EventKind {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "own goal"):
		return provider.EventOwnGoal
	case strings.Contains(t, "penalty") && (strings.Contains(t, "goal") || strings.Contains(t, "scored")):
		return provider.EventPenalty
	case strings.Contains(t, "goal"):
		return provider.EventGoal
	case strings.Contains(t, "yellow"):
		return provider.EventYellow
	case strings.Contains(t, "red"):
		return provider.EventRed
	case strings.Contains(t, "substitution"):
		return provider.EventSub
	default:
		return provider.EventOther
	}
}

// groupByTeam builds an abbreviation→group-letter map from standings. It is
// best-effort: any failure yields an empty map rather than an error, since
// group labels only decorate the output.
func (c *Client) groupByTeam(ctx context.Context) map[string]string {
	out := map[string]string{}
	groups, err := c.Standings(ctx)
	if err != nil {
		return out
	}
	for _, g := range groups {
		for _, s := range g.Standings {
			out[s.Team.Abbr] = g.Letter
		}
	}
	return out
}

func toMatch(e event, round string, groups map[string]string) (provider.Match, bool) {
	if len(e.Competitions) == 0 {
		return provider.Match{}, false
	}
	comp := e.Competitions[0]
	var home, away competitor
	var haveHome, haveAway bool
	for _, c := range comp.Competitors {
		switch c.HomeAway {
		case "home":
			home, haveHome = c, true
		case "away":
			away, haveAway = c, true
		}
	}
	if !haveHome || !haveAway {
		return provider.Match{}, false
	}

	kick, _ := parseTime(e.Date)
	st := comp.Status
	if st.Type.State == "" {
		st = e.Status
	}

	m := provider.Match{
		ID:        e.ID,
		Kick:      kick,
		State:     toState(st.Type.State),
		Home:      provider.Team{Abbr: home.Team.Abbreviation, Name: home.Team.DisplayName, Group: groups[home.Team.Abbreviation]},
		Away:      provider.Team{Abbr: away.Team.Abbreviation, Name: away.Team.DisplayName, Group: groups[away.Team.Abbreviation]},
		HomeScore: atoi(home.Score),
		AwayScore: atoi(away.Score),
		Winner:    winnerSide(home, away),
		Shootout:  home.ShootoutScore != nil || away.ShootoutScore != nil,
		HomeShootout: derefInt(home.ShootoutScore),
		AwayShootout: derefInt(away.ShootoutScore),
		Detail:    st.Type.ShortDetail,
		Clock:     st.DisplayClock,
		Venue:     comp.Venue.FullName,
		Round:     round,
		Group:     groups[home.Team.Abbreviation],
	}
	return m, true
}

// winnerSide reports which side the source flagged as the winner, or "" when
// neither is flagged (draw, or match not yet decided).
func winnerSide(home, away competitor) string {
	switch {
	case home.Winner:
		return "home"
	case away.Winner:
		return "away"
	default:
		return ""
	}
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// clampGroup drops the group letter from a knockout match. The group label is
// derived from the teams' group membership, which stays set after the group
// stage — so without this a Round-of-32 fixture would mislabel itself as a
// group game. Left untouched when the round is unknown (empty) so a failed
// round lookup degrades to showing the group rather than nothing.
func clampGroup(m *provider.Match) {
	if m.Round != "" && !isGroupStage(m.Round) {
		m.Group = ""
	}
}

func isGroupStage(round string) bool {
	return strings.Contains(strings.ToLower(round), "group")
}

func toState(s string) provider.MatchState {
	switch s {
	case "in":
		return provider.StateLive
	case "post":
		return provider.StateFinished
	default:
		return provider.StateScheduled
	}
}

// parseTime accepts ESPN's timestamps, which may omit seconds
// ("2026-06-16T19:00Z") or include them ("2026-06-16T19:00:00Z").
func parseTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02T15:04Z07:00", "2006-01-02T15:04:05Z0700"}
	var err error
	var t time.Time
	for _, l := range layouts {
		if t, err = time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// groupLetter extracts "A" from "Group A".
func groupLetter(name string) string {
	if len(name) == 0 {
		return ""
	}
	// Names look like "Group A"; take the last whitespace-separated token.
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == ' ' {
			return name[i+1:]
		}
	}
	return name
}
