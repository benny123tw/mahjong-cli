package game

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// Events log every state transition. Golden-game tests serialize the slice
// to JSON and diff against a frozen snapshot under testdata/game/golden/;
// the TUI consumes the same slice for animation cues.
//
// EventNop in state.go is a placeholder Step return that says "something
// happened, look at State() for the consequence" — the typed events here
// are richer payloads emitted alongside state mutations.

// EventDeal — wall constructed and dealt; one per round at New().
type EventDeal struct {
	Hands         [numSeats][]tile.Tile
	DoraIndicator tile.Tile
}

// EventDraw — Player drew Tile from the live wall.
type EventDraw struct {
	Player Seat
	Tile   tile.Tile
}

// EventDiscard — Player discarded Tile from index Index of their hand.
type EventDiscard struct {
	Player Seat
	Tile   tile.Tile
	Index  int
}

// EventCall — Player called Kind on Discarder's tile, forming Meld.
type EventCall struct {
	Player    Seat
	Discarder Seat
	Kind      ClaimKind
	Meld      Meld
}

// EventWin — Winner won by Kind (ron or tsumo) on Tile, scored as Result.
type EventWin struct {
	Winner Seat
	Loser  Seat
	Kind   ClaimKind
	Tile   tile.Tile
	Hand   hand.Hand
	Result *calc.Result
}

// EventRoundEnd — round terminated with the given Outcome.
type EventRoundEnd struct {
	Outcome Outcome
}

func (EventDeal) isEvent()     {}
func (EventDraw) isEvent()     {}
func (EventDiscard) isEvent()  {}
func (EventCall) isEvent()     {}
func (EventWin) isEvent()      {}
func (EventRoundEnd) isEvent() {}
