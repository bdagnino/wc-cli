// Package fifa resolves the official FIFA highlights video for a finished
// match. It uses the same public, no-auth endpoints fifa.com itself calls:
// the season calendar (to map a team-code pair to FIFA's internal match ids)
// and the match-details videos section (to read the highlights page URL).
//
// It is intentionally independent of provider.Provider: highlights are a
// best-effort decoration layered on top of the core tournament data, and the
// caller falls back to a search link when this returns nothing.
package fifa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// World Cup 2026 identifiers. This CLI targets a single tournament, so these
// are fixed (mirroring the hardcoded "fifa.world" the ESPN client uses).
const (
	competitionID = "17"
	seasonID      = "285023"
)

const (
	defaultAPIBase = "https://api.fifa.com/api/v3"
	defaultCXMBase = "https://cxm-api.fifa.com/fifaplusweb/api"
	siteBase       = "https://www.fifa.com"
)

// Client looks up FIFA highlights links. The zero value is not usable; use New.
type Client struct {
	HTTP *http.Client
	// Endpoint roots, overridable in tests.
	apiBase string
	cxmBase string

	// The calendar is fetched once and memoized: a single run may resolve
	// several matches, and the mapping is stable within a run.
	once     sync.Once
	index    map[string][]matchRef
	indexErr error
}

// matchRef is one calendar match: enough to disambiguate a code pair and to
// address the videos endpoint.
type matchRef struct {
	home, away           string
	homeScore, awayScore int
	stageID, matchID     string
}

// New returns a Client with sensible network defaults.
func New() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 15 * time.Second},
		apiBase: defaultAPIBase,
		cxmBase: defaultCXMBase,
	}
}

// Highlights returns the official FIFA highlights page URL for the match
// between two team codes (FIFA/ESPN share the same 3-letter codes), or ""
// when none is published or the match can't be resolved. Scores disambiguate
// the rare case where the same pair meets twice (group stage + a later round).
// Errors are returned for genuine network/parse failures so the caller can log
// them; a clean "no highlights" is ("", nil).
func (c *Client) Highlights(ctx context.Context, homeCode, awayCode string, homeScore, awayScore int) (string, error) {
	ref, ok := c.lookup(ctx, homeCode, awayCode, homeScore, awayScore)
	if c.indexErr != nil {
		return "", c.indexErr
	}
	if !ok {
		return "", nil
	}
	return c.highlightURL(ctx, ref)
}

// lookup resolves a code pair (and scores, as a tiebreak) to its calendar
// match, loading and memoizing the calendar on first use.
func (c *Client) lookup(ctx context.Context, homeCode, awayCode string, hs, as int) (matchRef, bool) {
	c.once.Do(func() { c.index, c.indexErr = c.buildIndex(ctx) })
	if c.indexErr != nil {
		return matchRef{}, false
	}
	refs := c.index[pairKey(homeCode, awayCode)]
	switch len(refs) {
	case 0:
		return matchRef{}, false
	case 1:
		return refs[0], true
	}
	// Disambiguate by matching the exact scoreline in the right orientation.
	for _, r := range refs {
		if strings.EqualFold(r.home, homeCode) && r.homeScore == hs && r.awayScore == as {
			return r, true
		}
	}
	return refs[0], true
}

// buildIndex fetches the season calendar and maps each match by its
// order-independent team-code pair.
func (c *Client) buildIndex(ctx context.Context) (map[string][]matchRef, error) {
	url := fmt.Sprintf("%s/calendar/matches?language=en&idCompetition=%s&idSeason=%s&count=400",
		c.apiBase, competitionID, seasonID)
	var raw calendarResp
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	idx := make(map[string][]matchRef, len(raw.Results))
	for _, m := range raw.Results {
		home, away := m.Home.IdCountry, m.Away.IdCountry
		if home == "" || away == "" {
			continue // unfilled knockout slots
		}
		key := pairKey(home, away)
		idx[key] = append(idx[key], matchRef{
			home:      home,
			away:      away,
			homeScore: derefInt(m.HomeTeamScore),
			awayScore: derefInt(m.AwayTeamScore),
			stageID:   m.IdStage,
			matchID:   m.IdMatch,
		})
	}
	return idx, nil
}

// highlightURL calls the match-details videos section and returns the absolute
// URL of the Highlights item, or "" when none is present.
func (c *Client) highlightURL(ctx context.Context, ref matchRef) (string, error) {
	url := fmt.Sprintf("%s/sections/matchdetails/videos?locale=en&competitionId=%s&seasonId=%s&stageId=%s&matchId=%s",
		c.cxmBase, competitionID, seasonID, ref.stageID, ref.matchID)
	var raw videosResp
	if err := c.get(ctx, url, &raw); err != nil {
		return "", err
	}
	for _, it := range raw.VodVideosBaseCarousel.Items {
		if strings.EqualFold(it.VideoSubcategory, "Highlights") && it.ReadMorePageURL != "" {
			return siteBase + it.ReadMorePageURL, nil
		}
	}
	return "", nil
}

func (c *Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("reaching FIFA: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FIFA returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding FIFA response: %w", err)
	}
	return nil
}

// pairKey is an order-independent key for two team codes.
func pairKey(a, b string) string {
	a, b = strings.ToUpper(a), strings.ToUpper(b)
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0] + "|" + pair[1]
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
