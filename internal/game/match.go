package game

import (
	"errors"
	"slices"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// hanchanLength is the number of hands in a standard hanchan: East 1-4 + South 1-4.
const hanchanLength = 8

// ErrMatchAlreadyFinished is returned by Match.AdvanceFromOutcome when the
// match has already terminated (hanchan complete or tobi).
var ErrMatchAlreadyFinished = errors.New("game: match already finished")

// MatchOutcome records why a match terminated. Reason is a short string
// ("hanchan-complete" or "tobi"); BustSeat is populated only for tobi.
type MatchOutcome struct {
	Reason   string
	BustSeat Seat
}

// TransitionResult is the per-hand summary returned by AdvanceFromOutcome
// for the TUI's end-of-hand acknowledgement panel.
type TransitionResult struct {
	Deltas       [4]int
	NewTotals    [4]int
	Renchan      bool
	NewHandIndex int
	NewHonba     int
	MatchOutcome *MatchOutcome
}

// MatchOptions configures hanchan-level rules that propagate to every
// per-hand *Game. Akadora is the only knob today; the struct gives room
// for future toggles (kuitan, atozuke, etc.) without further constructor
// proliferation.
type MatchOptions struct {
	Akadora bool
}

// Match owns hanchan-level state across multiple per-hand Games. The
// per-hand Game (CurrentGame()) is rebuilt by AdvanceFromOutcome on each
// transition, with seat winds rotated dealer-relative and per-hand seed
// derived from the base match seed plus hand index.
type Match struct {
	scores       [numSeats]int
	dealer       Seat
	roundWind    uint8
	handIndex    int
	honba        int
	riichiSticks int
	seed         int64
	opts         MatchOptions
	currentGame  *Game
	outcome      *MatchOutcome
}

// NewMatch starts a fresh hanchan with akadora enabled (modern default):
// all seats at 25000, dealer = SeatEast, round wind East, hand index 0
// (East 1), honba 0, no riichi sticks. Callers needing akadora-off should
// use NewMatchWithOptions.
func NewMatch(seed int64) *Match {
	return NewMatchWithOptions(seed, MatchOptions{Akadora: true})
}

// NewMatchWithOptions starts a fresh hanchan threading MatchOptions through
// to the per-hand *Game (and from there to the wall). The opts are stored
// on the Match so subsequent AdvanceFromOutcome rebuilds keep the same rule.
func NewMatchWithOptions(seed int64, opts MatchOptions) *Match {
	m := &Match{
		dealer:    SeatEast,
		roundWind: tile.EastWind,
		seed:      seed,
		opts:      opts,
	}
	for s := range Seat(numSeats) {
		m.scores[s] = 25000
	}
	m.currentGame = NewWithDealerOptions(
		seed,
		SeatEast,
		tile.EastWind,
		GameOptions(opts),
	)
	return m
}

// Scores returns a defensive copy of per-seat point totals.
func (m *Match) Scores() [numSeats]int { return m.scores }

// Dealer returns the seat with East-wind for the current hand.
func (m *Match) Dealer() Seat { return m.dealer }

// RoundWind returns the round wind for the current hand (East 1-4 = East,
// South 1-4 = South).
func (m *Match) RoundWind() uint8 { return m.roundWind }

// HandIndex returns the 0-indexed hand number (0 = East 1, 7 = South 4).
func (m *Match) HandIndex() int { return m.handIndex }

// Honba returns the consecutive renchan counter for this hand index.
func (m *Match) Honba() int { return m.honba }

// RiichiSticks returns the pooled stick count (1000 per stick). Swept by
// the next agari winner.
func (m *Match) RiichiSticks() int { return m.riichiSticks }

// CurrentGame returns the per-hand state machine. After IsFinished returns
// true, this points at the just-finished hand for inspection.
func (m *Match) CurrentGame() *Game { return m.currentGame }

// IsFinished reports whether the hanchan has terminated.
func (m *Match) IsFinished() bool { return m.outcome != nil }

// FinalOutcome returns the match terminator, or nil while the match is live.
func (m *Match) FinalOutcome() *MatchOutcome { return m.outcome }

// HandLabel returns the human-readable hand name ("East 1" .. "South 4").
func (m *Match) HandLabel() string {
	round := "East"
	if m.handIndex >= 4 {
		round = "South"
	}
	hand := (m.handIndex % 4) + 1
	return round + " " + string(rune('0'+hand))
}

// SetTestScore overrides a seat's score directly. Test-only — used by
// match_test.go fixtures that need a near-tobi score without driving a
// full hand.
func (m *Match) SetTestScore(s Seat, points int) { m.scores[s] = points }

// SetTestRiichiSticks overrides the pooled-stick count. Test-only.
func (m *Match) SetTestRiichiSticks(n int) { m.riichiSticks = n }

// SetTestHandIndex jumps the match to a specific hand index for test setup.
// The dealer and round wind are recomputed (dealer = (handIndex % 4) East,
// round = East for 0-3, South for 4-7).
func (m *Match) SetTestHandIndex(idx int) {
	m.handIndex = idx
	m.dealer = Seat(idx % numSeats)
	if idx < 4 {
		m.roundWind = tile.EastWind
	} else {
		m.roundWind = tile.SouthWind
	}
	m.currentGame = NewWithDealerOptions(
		m.seed+int64(idx),
		m.dealer,
		m.roundWind,
		GameOptions(m.opts),
	)
}

// SetTestHonba overrides the honba counter for fixture setup.
func (m *Match) SetTestHonba(n int) { m.honba = n }

// AdvanceFromOutcome consumes the active hand's terminal outcome,
// computes payouts, applies them to scores, and prepares the next hand
// (or finishes the match). The returned TransitionResult is the TUI's
// end-of-hand summary payload.
func (m *Match) AdvanceFromOutcome(o Outcome) (TransitionResult, error) {
	if m.outcome != nil {
		return TransitionResult{}, ErrMatchAlreadyFinished
	}

	// Pool this hand's riichi deposits BEFORE payouts so the winner sweeps
	// every stick on the table — including their own, if they declared
	// and won. The per-hand Game already deducted 1000 from each declarer
	// at declaration time; here we tally the sticks themselves.
	if m.currentGame != nil {
		for s := range Seat(numSeats) {
			if m.currentGame.riichiDeclared[s] {
				m.riichiSticks++
			}
		}
	}

	ctx := PayoutContext{Dealer: m.dealer, Honba: m.honba, RiichiSticks: m.riichiSticks}
	deltas := ComputePayouts(o, ctx)
	for s := range Seat(numSeats) {
		m.scores[s] += deltas[s]
	}

	// Agari sweeps the pool; ryuukyoku leaves it for the next agari winner.
	switch o.(type) {
	case OutcomeRon, OutcomeTsumo:
		m.riichiSticks = 0
	}

	renchan := m.isRenchan(o)

	tr := TransitionResult{
		Deltas:  deltas,
		Renchan: renchan,
	}

	if renchan {
		m.honba++
	} else {
		m.handIndex++
		m.honba = 0
		m.dealer = m.dealer.Next()
		if m.handIndex == 4 {
			m.roundWind = tile.SouthWind
		}
	}

	// Tobi check: any seat below 0 ends the match immediately.
	for s := range Seat(numSeats) {
		if m.scores[s] < 0 {
			m.outcome = &MatchOutcome{Reason: "tobi", BustSeat: s}
			tr.MatchOutcome = m.outcome
			tr.NewTotals = m.scores
			tr.NewHandIndex = m.handIndex
			tr.NewHonba = m.honba
			return tr, nil
		}
	}

	// Hanchan completion: handIndex past South 4 with no renchan.
	if m.handIndex >= hanchanLength {
		m.outcome = &MatchOutcome{Reason: "hanchan-complete"}
		tr.MatchOutcome = m.outcome
		tr.NewTotals = m.scores
		tr.NewHandIndex = m.handIndex
		tr.NewHonba = m.honba
		return tr, nil
	}

	// Build the next hand's Game.
	m.currentGame = NewWithDealerOptions(
		m.seed+int64(m.handIndex),
		m.dealer,
		m.roundWind,
		GameOptions(m.opts),
	)
	tr.NewTotals = m.scores
	tr.NewHandIndex = m.handIndex
	tr.NewHonba = m.honba
	return tr, nil
}

// isRenchan returns true when the outcome implies the dealer keeps the
// dealership: dealer-win on tsumo or ron, OR dealer-tenpai at exhaustive
// ryuukyoku.
func (m *Match) isRenchan(o Outcome) bool {
	switch v := o.(type) {
	case OutcomeRon:
		return v.Winner == m.dealer
	case OutcomeTsumo:
		return v.Winner == m.dealer
	case OutcomeRyuukyoku:
		return slices.Contains(v.TenpaiPlayers, m.dealer)
	}
	return false
}
