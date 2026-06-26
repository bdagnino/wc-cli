package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bdagnino/wc-cli/internal/provider"
)

// Card / column geometry for the full bracket. Cards carry the kickoff date and
// two 3-letter codes; the date drives the minimum width. The whole thing is
// laid onto a rune canvas so the connectors between rounds line up exactly.
const (
	cardW   = 16
	cardH   = 4
	labelW  = 9 // team code column inside a card
	scoreW  = 2
	pitch   = 5 // rows per leaf (card height + 1 gap)
	connGap = 3 // horizontal gap between columns, for connectors
	colW    = cardW + connGap
	flagW   = 2 // display columns a flag emoji occupies
)

func colX(r bRound) int { return int(r) * colW }
func iround(f float64) int { return int(math.Round(f)) }

// Style ids on the canvas, resolved to lipgloss styles at render time so we can
// colour individual cells without juggling ANSI inside the grid.
const (
	stNone uint8 = iota
	stFaint
	stMuted
	stTeam
	stWin
	stTitle
	stLive
	stProj
)

func paint(id uint8, s string) string {
	switch id {
	case stFaint:
		return Faint.Render(s)
	case stMuted:
		return Muted.Render(s)
	case stTeam:
		return Header.Render(s)
	case stWin:
		return Winner.Render(s)
	case stTitle:
		return Title.Render(s)
	case stLive:
		return Live.Render(s)
	case stProj:
		return Pencil.Render(s)
	}
	return s
}

// cell holds a glyph string (usually one rune, but a flag emoji is several) so
// the canvas can carry double-width glyphs. A wide glyph lives in its left cell;
// the columns it spills into are marked cont and emit nothing.
type cell struct {
	g    string
	s    uint8
	cont bool
}

func (c cell) blank() bool { return c.g == "" && !c.cont && c.s == stNone }

type canvas struct {
	w, h int
	g    [][]cell
}

func newCanvas(w, h int) *canvas {
	g := make([][]cell, h)
	for y := range g {
		g[y] = make([]cell, w)
	}
	return &canvas{w, h, g}
}

func (c *canvas) put(x, y int, r rune, s uint8) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.g[y][x] = cell{g: string(r), s: s}
	}
}

// putWide places a glyph that occupies width display columns (a flag is 2),
// reserving the spilled-into columns so everything to its right stays aligned.
func (c *canvas) putWide(x, y int, glyph string, s uint8, width int) {
	if x < 0 || x >= c.w || y < 0 || y >= c.h {
		return
	}
	c.g[y][x] = cell{g: glyph, s: s}
	for k := 1; k < width && x+k < c.w; k++ {
		c.g[y][x+k] = cell{cont: true, s: s}
	}
}

func (c *canvas) text(x, y int, str string, s uint8) {
	for i, r := range []rune(str) {
		c.put(x+i, y, r, s)
	}
}

func (c *canvas) String() string {
	var b strings.Builder
	for y := 0; y < c.h; y++ {
		row := c.g[y]
		end := c.w
		for end > 0 && row[end-1].blank() {
			end--
		}
		for i := 0; i < end; {
			if row[i].cont { // spilled column of a wide glyph already emitted
				i++
				continue
			}
			var seg strings.Builder
			j := i
			for j < end && row[j].s == row[i].s {
				if !row[j].cont {
					if row[j].g == "" {
						seg.WriteByte(' ')
					} else {
						seg.WriteString(row[j].g)
					}
				}
				j++
			}
			b.WriteString(paint(row[i].s, seg.String()))
			i = j
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// layout assigns each match a vertical center: leaves stack at a fixed pitch,
// and every parent sits at the midpoint of its two feeders.
func (b *Bracket) layout() {
	leaf := 0
	var walk func(m *bMatch) float64
	walk = func(m *bMatch) float64 {
		if m == nil {
			return 0
		}
		if m.round == rR32 || (m.upper == nil && m.lower == nil) {
			c := float64(leaf)*pitch + 2
			leaf++
			m.center = c
			return c
		}
		var cs []float64
		if m.upper != nil {
			cs = append(cs, walk(m.upper))
		}
		if m.lower != nil {
			cs = append(cs, walk(m.lower))
		}
		if len(cs) == 0 {
			c := float64(leaf)*pitch + 2
			leaf++
			m.center = c
			return c
		}
		m.center = (cs[0] + cs[len(cs)-1]) / 2
		return m.center
	}
	walk(b.final)
}

// Render draws the full knockout bracket. Returns "" when there is no bracket.
func (b *Bracket) Render(loc *time.Location) string {
	leaves := len(b.rounds[rR32])
	if b.final == nil || leaves == 0 {
		return ""
	}
	b.layout()
	w := int(nRounds-1)*colW + cardW
	h := leaves*pitch + 1
	cv := newCanvas(w, h)

	for r := rR16; r < nRounds; r++ {
		for _, m := range b.rounds[r] {
			b.drawConnector(cv, m)
		}
	}
	for r := bRound(0); r < nRounds; r++ {
		for _, m := range b.rounds[r] {
			drawCard(cv, m, loc)
		}
	}

	header := newCanvas(w, 1)
	for r := bRound(0); r < nRounds; r++ {
		header.text(colX(r), 0, fit(roundTitles[r], cardW), stTitle)
	}
	return header.String() + cv.String()
}

func drawCard(cv *canvas, m *bMatch, loc *time.Location) {
	x := colX(m.round)
	cy := iround(m.center)
	top := cy - 2

	// Top border carries the date.
	date := m.kick.In(loc).Format("2 Jan 15:04")
	tb := []rune("╭─ " + date + " ")
	for len(tb) < cardW-1 {
		tb = append(tb, '─')
	}
	tb = tb[:cardW-1]
	tb = append(tb, '╮')
	cv.text(x, top, string(tb), stFaint)

	drawSide(cv, x, top+1, m, true)
	drawSide(cv, x, top+2, m, false)

	cv.text(x, top+3, "╰"+strings.Repeat("─", cardW-2)+"╯", stFaint)
}

func drawSide(cv *canvas, x, y int, m *bMatch, home bool) {
	slot := m.away
	score := m.aScore
	if home {
		slot = m.home
		score = m.hScore
	}
	finished := m.state == provider.StateFinished
	won := finished && ((home && m.hScore > m.aScore) || (!home && m.aScore > m.hScore))

	id := stTeam
	switch {
	case slot.projected:
		id = stProj
	case !slot.real:
		id = stMuted
	}
	if finished {
		if won {
			id = stWin
		} else {
			id = stMuted
		}
	} else if m.state == provider.StateLive {
		id = stLive
	}

	cv.put(x, y, '│', stFaint)
	cv.put(x+1, y, ' ', stFaint)
	// Flag (2 columns) for real or penciled-in teams; blank for bare
	// placeholders, so codes align.
	if slot.real || slot.projected {
		cv.putWide(x+2, y, Flag(slot.abbr), id, 2)
	}
	// Then a gap and the short code/token in the remaining label width.
	cv.text(x+2+flagW+1, y, fitLeft(cardLabel(slot), labelW-flagW-1), id)
	cv.put(x+2+labelW, y, ' ', stFaint)
	sc := ""
	if m.state != provider.StateScheduled {
		sc = strconv.Itoa(score)
	}
	cv.text(x+2+labelW+1, y, fitRight(sc, scoreW), id)
	cv.put(cardW+x-1, y, '│', stFaint)
}

// cardLabel is the short code shown in a card: a real team's 3-letter code, a
// group placeholder's token ("1I", "3RD"), or "TBD" for an undecided winner.
func cardLabel(s bSlot) string {
	if s.real {
		return s.abbr
	}
	if s.hasSrc || s.abbr == "" {
		return "TBD"
	}
	return s.abbr
}

func (b *Bracket) drawConnector(cv *canvas, m *bMatch) {
	cr := m.round - 1
	gapStart := colX(cr) + cardW
	midX := gapStart + connGap/2
	parentLeft := colX(m.round)
	rp := iround(m.center)

	var rows []int
	if m.upper != nil {
		rows = append(rows, iround(m.upper.center))
	}
	if m.lower != nil {
		rows = append(rows, iround(m.lower.center))
	}
	if len(rows) == 0 {
		return
	}
	for _, ry := range rows {
		for x := gapStart; x < midX; x++ {
			cv.put(x, ry, '─', stFaint)
		}
	}
	top, bot := rows[0], rows[len(rows)-1]
	if top > bot {
		top, bot = bot, top
	}
	for y := top + 1; y < bot; y++ {
		cv.put(midX, y, '│', stFaint)
	}
	if len(rows) == 2 {
		cv.put(midX, rows[0], '┐', stFaint)
		cv.put(midX, rows[1], '┘', stFaint)
	}
	cv.put(midX, rp, '├', stFaint)
	for x := midX + 1; x < parentLeft; x++ {
		cv.put(x, rp, '─', stFaint)
	}
}

// Path renders one team's route from the Round of 32 to the final. ok is false
// when the team isn't found among the knockout fixtures.
func (b *Bracket) Path(query string, loc *time.Location) (string, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	var start *bMatch
	var side int
	for _, m := range b.rounds[rR32] {
		if (m.home.real || m.home.projected) && slotMatches(m.home, q) {
			start, side = m, 0
			break
		}
		if (m.away.real || m.away.projected) && slotMatches(m.away, q) {
			start, side = m, 1
			break
		}
	}
	if start == nil {
		return "", false
	}
	teamSlot := start.home
	if side == 1 {
		teamSlot = start.away
	}

	type step struct {
		m    *bMatch
		side int
	}
	steps := []step{{start, side}}
	cur := start
	for cur.round < rFinal {
		par, ps := b.parentOf(cur)
		if par == nil {
			break
		}
		steps = append(steps, step{par, ps})
		cur = par
	}

	var b2 strings.Builder
	nameStyle := Header
	if teamSlot.projected {
		nameStyle = Pencil
	}
	title := Flag(teamSlot.abbr) + " " + nameStyle.Render(teamSlot.name) + Faint.Render(" — road to the final")
	b2.WriteString(title + "\n\n")
	for _, s := range steps {
		opp := s.m.away
		if s.side == 1 {
			opp = s.m.home
		}
		round := Muted.Width(14).Render(roundTitles[s.m.round])
		date := Faint.Render(fitLeft(s.m.kick.In(loc).Format("Mon 2 Jan 15:04"), 16))
		us := Flag(teamSlot.abbr) + " " + Header.Render(teamSlot.abbr)
		vs := Faint.Render(" vs ")
		b2.WriteString("  " + round + "  " + date + "  " + us + vs + opponentLabel(opp) + "\n")
	}
	return b2.String(), true
}

// parentOf finds the match a given match feeds into, and which side (0 home, 1
// away) it lands on. It scans subsequent rounds rather than only the next one,
// so a non-contiguous bracket still resolves.
func (b *Bracket) parentOf(m *bMatch) (*bMatch, int) {
	for rr := m.round + 1; rr <= rFinal; rr++ {
		for _, p := range b.rounds[rr] {
			if p.upper == m {
				return p, 0
			}
			if p.lower == m {
				return p, 1
			}
		}
	}
	return nil, 0
}

func slotMatches(s bSlot, q string) bool {
	return strings.EqualFold(s.abbr, q) || strings.Contains(strings.ToLower(s.name), q)
}

// opponentLabel humanizes the other side of a match in the path view: a real
// team with its flag, a group placeholder verbatim, or the feeder it awaits.
func opponentLabel(s bSlot) string {
	if s.real {
		return Flag(s.abbr) + " " + Header.Render(s.name)
	}
	if s.projected {
		return Flag(s.abbr) + " " + Pencil.Render(s.name)
	}
	if s.hasSrc {
		return Muted.Render(fmt.Sprintf("winner of %s #%d", shortRound(s.srcRound), s.srcN))
	}
	if s.name != "" {
		return Muted.Render(s.name)
	}
	return Muted.Render("TBD")
}

func shortRound(r bRound) string {
	switch r {
	case rR32:
		return "R32"
	case rR16:
		return "R16"
	case rQF:
		return "QF"
	case rSF:
		return "SF"
	default:
		return "F"
	}
}

// fit pads or truncates s to exactly w runes (left-aligned).
func fit(s string, w int) string { return fitLeft(s, w) }

func fitLeft(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func fitRight(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return strings.Repeat(" ", w-len(r)) + s
}
