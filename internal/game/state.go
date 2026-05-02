package game

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// Seat identifies one of the four players, in physical turn order:
// East draws first, then South, West, North.
type Seat uint8

const (
	SeatEast Seat = iota
	SeatSouth
	SeatWest
	SeatNorth
)

// HumanSeat is the seat the human player occupies. The engine maintains a
// canonical-sort invariant on this seat's concealed hand (see sortConcealed).
// Bot seats' hands are never sorted by the engine — bot decision logic is
// order-independent.
const HumanSeat = SeatSouth

// SeatWind returns the wind tile ID for this seat. East=27 .. North=30.
func (s Seat) SeatWind() uint8 {
	return tile.EastWind + uint8(s)
}

// Next returns the seat that draws after this one (counter-clockwise in
// physical seating, but clockwise in turn order: E → S → W → N → E).
func (s Seat) Next() Seat { return (s + 1) % numSeats }

// Kamicha returns the seat to this seat's left (the one whose turn would
// naturally come next). In riichi only the kamicha can be chi'd.
func (s Seat) Kamicha() Seat { return (s + numSeats - 1) % numSeats }

// State is the union of game-loop states. Concrete types embed it via the
// unexported isState() marker — go's only sealed-interface idiom.
type State interface{ isState() }

// StateAwaitingDraw — Player is about to draw a tile from the live wall.
type StateAwaitingDraw struct{ Player Seat }

// StateAwaitingDiscard — Player has just drawn (hand size 14) and must
// discard, or declare tsumo / riichi.
type StateAwaitingDiscard struct{ Player Seat }

// StateAwaitingClaims — Discarder just discarded `Discard`. Other players
// have a claim window. The state resolves to either AwaitingDiscard
// (claim winner takes the tile and discards) or AwaitingDraw (no claim,
// next player draws).
type StateAwaitingClaims struct {
	Discard   tile.Tile
	Discarder Seat
}

// StateRoundOver — round terminated by agari (tsumo / ron) or ryuukyoku.
// Outcome carries the details for scoring and event-log capture.
type StateRoundOver struct{ Outcome Outcome }

// StateGameOver — terminal state past the hanchan boundary. Single round
// in v1, so this is reached only when the (single) round completes.
type StateGameOver struct{}

// StateAwaitingChankan — Declarer has just submitted shouminkan; the
// engine pauses to allow other seats one ron claim on UpgradeTile before
// the kan completes. Only InputResolveClaims (ron-only) is honored.
type StateAwaitingChankan struct {
	UpgradeTile tile.Tile
	Declarer    Seat
}

func (StateAwaitingDraw) isState()    {}
func (StateAwaitingDiscard) isState() {}
func (StateAwaitingClaims) isState()  {}
func (StateAwaitingChankan) isState() {}
func (StateRoundOver) isState()       {}
func (StateGameOver) isState()        {}

// Outcome is the round-termination payload. Refined further in task 5.2 to
// include scored hand details and ryuukyoku tenpai bookkeeping.
type Outcome interface{ isOutcome() }

// OutcomeRyuukyoku — wall exhausted with no winner. TenpaiPlayers is the
// list of seats that were in tenpai at exhaust time (for noten payments).
type OutcomeRyuukyoku struct {
	TenpaiPlayers []Seat
}

// OutcomeRon — Winner won by ron on Loser's discarded Tile. Hand and Result
// hold the winning hand and its scored analysis (when scoring is wired —
// task 8.x for the TUI path; the v1 ron transition records bare seat info).
type OutcomeRon struct {
	Winner Seat
	Loser  Seat
	Tile   tile.Tile
	Hand   hand.Hand
	Result *calc.Result
}

// OutcomeTsumo — Winner won by tsumo on their drawn Tile.
type OutcomeTsumo struct {
	Winner Seat
	Tile   tile.Tile
	Hand   hand.Hand
	Result *calc.Result
}

func (OutcomeRyuukyoku) isOutcome() {}
func (OutcomeRon) isOutcome()       {}
func (OutcomeTsumo) isOutcome()     {}

// Input is the union of player-driven inputs the state machine accepts.
type Input interface{ isInput() }

// InputDraw — only valid in StateAwaitingDraw. Pops a tile from the live
// wall into the active player's hand.
type InputDraw struct{}

// InputDiscard — only valid in StateAwaitingDiscard. Removes the tile at
// `Index` from the active player's hand.
//
// When `Riichi` is true, the discard doubles as a riichi declaration. The
// engine validates four preconditions before accepting (concealed hand,
// ≥1000 points, ≥4 wall tiles remaining, post-discard tenpai); on failure
// it returns ErrIllegalRiichi and leaves state unchanged.
type InputDiscard struct {
	Index  int
	Riichi bool
}

// InputDeclareTsumo — only valid in StateAwaitingDiscard. Active player
// claims the just-drawn tile is the winning tile.
type InputDeclareTsumo struct{}

// InputResolveClaims — only valid in StateAwaitingClaims. Claims maps each
// claiming seat to the kind of claim being attempted; nil or empty Claims
// means no-one called and the next player draws.
type InputResolveClaims struct {
	Claims map[Seat]Claim
}

// InputDeclareAnkan — only valid in StateAwaitingDiscard. Declares a
// concealed kan when the active seat's hand contains exactly 4 tiles
// matching `TileID`. On success, the four tiles become a MeldKan with
// KanKind=KanAnkan, the engine reveals an additional dora indicator,
// pulls a rinshan replacement tile, and stays in AwaitingDiscard.
type InputDeclareAnkan struct {
	TileID uint8
}

// InputDeclareShouminkan — only valid in StateAwaitingDiscard. Upgrades
// an existing open MeldPon (matching `TileID`) by appending the 4th tile
// from the active seat's concealed hand. The engine transitions to
// StateAwaitingChankan to give other seats a ron window before the kan
// completes; on no ron, the pon is upgraded to KanShouminkan and the
// rinshan flow runs.
type InputDeclareShouminkan struct {
	TileID uint8
}

func (InputDraw) isInput()              {}
func (InputDiscard) isInput()           {}
func (InputDeclareTsumo) isInput()      {}
func (InputResolveClaims) isInput()     {}
func (InputDeclareAnkan) isInput()      {}
func (InputDeclareShouminkan) isInput() {}

// ClaimKind enumerates the claim types resolvable on a discard. Pass is
// included as a no-op to make the resolver's input shape symmetric.
type ClaimKind uint8

const (
	ClaimPass ClaimKind = iota
	ClaimChi
	ClaimPon
	ClaimKan
	ClaimRon
)

// Claim is a single player's response to a discard.
//
// ChiTiles names the two tiles in the claimant's hand to combine with the
// discarded tile to form the chi sequence. Required only for chi; pon and
// kan are unambiguous.
type Claim struct {
	Kind     ClaimKind
	ChiTiles [2]uint8
}

// Event is what Step returns: a description of what just happened, useful
// for the TUI and golden-game tests. Refined in task 5.2.
type Event interface{ isEvent() }

// EventNop — placeholder event so Step can return without commitments while
// the event log is fleshed out in task 5.2. Removed before ship.
type EventNop struct{}

func (EventNop) isEvent() {}
