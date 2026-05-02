package game

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// Game is the round-scoped state machine. All mutations route through Step;
// the State, Hand, and Discards methods are read-only views.
//
// The struct deliberately keeps no reference to anything UI-related. The TUI
// observes state via the public methods and submits Inputs via Step.
type Game struct {
	state          State
	wall           *Wall
	hands          [numSeats][]tile.Tile
	discards       [numSeats][]tile.Tile
	melds          [numSeats][]Meld
	doraIndicators []tile.Tile
	roundWind      uint8

	// callsHappened tracks whether any call interrupted the round, used by
	// Group C yaku detection (ippatsu, double riichi, chiihou).
	callsHappened bool

	// hasDiscarded[seat] flips true once the seat has made any discard, used
	// alongside !callsHappened to detect Tenhou / Chiihou (dealer or
	// non-dealer winning on their first uninterrupted draw).
	hasDiscarded [numSeats]bool

	// log accumulates a per-transition string trace consumed by
	// golden-game tests. Format: one line per state transition or
	// significant event, e.g. "draw East 3m" / "discard East 1m" /
	// "ron South from East on 5p" / "ryuukyoku tenpai=[South]".
	log []string

	seed int64

	// testOpen is a per-seat open flag set by SetTestOpen for tests that
	// want to plant an "open hand" without going through real call flow.
	testOpen [numSeats]bool

	// Per-seat riichi state. riichiDeclared flips true on successful
	// declaration; ippatsuLive tracks the open ippatsu window (closed by
	// any call from any seat or by the declarer's own next draw);
	// doubleRiichi flips true when declaration happens on the seat's
	// first uninterrupted intake (no prior discards, no prior calls).
	riichiDeclared [numSeats]bool
	ippatsuLive    [numSeats]bool
	doubleRiichi   [numSeats]bool

	// Per-seat point totals. Initialized to 25000 in New(). The riichi
	// deposit deducts 1000; ryuukyoku noten payments and agari payouts
	// adjust further (payout integration ships with smart-ai).
	scores [numSeats]int
}

// Meld is an opened meld (called pon, chi, or kan). A future change adds
// kan support; for v1 only pon and chi are produced.
type Meld struct {
	Kind  MeldKind
	Tiles []tile.Tile
	From  Seat // discarder whose tile completed the meld
}

// MeldKind is the call type that produced this meld.
type MeldKind uint8

const (
	MeldPon MeldKind = iota
	MeldChi
	MeldKan
)

// New starts a fresh round: shuffles a 136-tile wall from `seed`, deals 13
// to each seat, reveals one dora indicator, and sets state to
// StateAwaitingDraw{East} per the dealer-draws-first rule.
func New(seed int64) *Game {
	w := NewWall(seed)
	deal := w.Deal()
	g := &Game{
		seed:           seed,
		wall:           w,
		state:          StateAwaitingDraw{Player: SeatEast},
		doraIndicators: []tile.Tile{deal.DoraIndicator},
		roundWind:      tile.EastWind,
	}
	for seat := range numSeats {
		g.hands[seat] = deal.Hands[seat]
		g.scores[seat] = 25000
	}
	sortConcealed(g.hands[HumanSeat])
	g.logf("deal seed=%d dora=%s", seed, deal.DoraIndicator)
	return g
}

// sortConcealed sorts a tile slice in-place by ascending tile-ID, which is
// the canonical riichi order: M1..M9, P1..P9, S1..S9, EastWind, SouthWind,
// WestWind, NorthWind, Haku, Hatsu, Chun. The tile package's iota-defined
// IDs are already laid out in this order, so a stable ID sort is canonical.
//
// Stable so tied IDs (e.g., two of the same tile, or a red five vs a normal
// five sharing an ID) keep their relative order — useful when the player
// has just slotted a freshly-drawn tile into an existing pair.
func sortConcealed(tiles []tile.Tile) {
	slices.SortStableFunc(tiles, func(a, b tile.Tile) int {
		return cmp.Compare(a.ID, b.ID)
	})
}

// Seed returns the seed used to construct this game. Useful for printing
// and reproducing in bug reports.
func (g *Game) Seed() int64 { return g.seed }

// EventLog returns the accumulated transition log. The slice is owned by
// the caller; mutating it does not affect future logs.
func (g *Game) EventLog() []string {
	out := make([]string, len(g.log))
	copy(out, g.log)
	return out
}

func (g *Game) logf(format string, args ...any) {
	g.log = append(g.log, fmt.Sprintf(format, args...))
}

// State returns the current state. The returned value is a copy of the union
// type and safe to type-switch.
func (g *Game) State() State { return g.state }

// Hand returns a defensive copy of the seat's concealed tiles.
func (g *Game) Hand(s Seat) []tile.Tile {
	out := make([]tile.Tile, len(g.hands[s]))
	copy(out, g.hands[s])
	return out
}

// Discards returns a defensive copy of the seat's pond.
func (g *Game) Discards(s Seat) []tile.Tile {
	out := make([]tile.Tile, len(g.discards[s]))
	copy(out, g.discards[s])
	return out
}

// Melds returns a defensive copy of the seat's open melds.
func (g *Game) Melds(s Seat) []Meld {
	out := make([]Meld, len(g.melds[s]))
	copy(out, g.melds[s])
	return out
}

// Wall exposes the underlying wall (for shanten/machi queries that need to
// know live-remaining counts; the TUI never mutates).
func (g *Game) Wall() *Wall { return g.wall }

// RoundWind returns the round-wind tile ID. East-only round in v1.
func (g *Game) RoundWind() uint8 { return g.roundWind }

// DoraIndicators returns the revealed dora indicator tiles.
func (g *Game) DoraIndicators() []tile.Tile {
	out := make([]tile.Tile, len(g.doraIndicators))
	copy(out, g.doraIndicators)
	return out
}

// Step advances the state machine in response to an input. Returns
// ErrInvalidInputForState when the input doesn't apply to the current state.
func (g *Game) Step(in Input) (Event, error) {
	switch s := g.state.(type) {
	case StateAwaitingDraw:
		return g.stepFromAwaitingDraw(s, in)
	case StateAwaitingDiscard:
		return g.stepFromAwaitingDiscard(s, in)
	case StateAwaitingClaims:
		return g.stepFromAwaitingClaims(s, in)
	case StateRoundOver, StateGameOver:
		return nil, ErrRoundOver
	}
	return nil, fmt.Errorf("game: unhandled state %T", g.state)
}

// ErrInvalidInputForState is returned when an input is submitted to a state
// that does not accept it (e.g., InputDraw while in AwaitingClaims).
var ErrInvalidInputForState = errors.New("game: invalid input for current state")

// ErrRoundOver is returned when any input is submitted after the round has
// already terminated. Callers should restart with New() to begin a new round.
var ErrRoundOver = errors.New("game: round already over")

// ErrIllegalDiscard is returned when InputDiscard.Index is out of range.
var ErrIllegalDiscard = errors.New("game: illegal discard index")

// ErrYakulessWin is returned when InputDeclareTsumo / InputDeclareRon would
// award a winning shape with no yaku — the spec's "yakuless win is not
// allowed" rule keeps the player in their current state.
var ErrYakulessWin = errors.New("game: winning shape has no yaku")

// ErrIllegalRiichi is returned when InputDiscard{Riichi: true} fails any of
// the four preconditions: hand is open, score < 1000, wall has < 4 tiles,
// or post-discard hand is not tenpai.
var ErrIllegalRiichi = errors.New("game: illegal riichi declaration")

// ErrFuritenRon is returned when a human ron claim is rejected because
// the seat's machi tile appears in their own pond (permanent furiten).
var ErrFuritenRon = errors.New("game: ron blocked by furiten")

func (g *Game) stepFromAwaitingDraw(s StateAwaitingDraw, in Input) (Event, error) {
	if _, ok := in.(InputDraw); !ok {
		return nil, ErrInvalidInputForState
	}
	t, ok := g.wall.Draw()
	if !ok {
		tenpai := g.tenpaiSeats()
		g.logf("ryuukyoku tenpai=%v", seatNames(tenpai))
		g.state = StateRoundOver{Outcome: OutcomeRyuukyoku{TenpaiPlayers: tenpai}}
		return EventNop{}, nil
	}
	g.hands[s.Player] = append(g.hands[s.Player], t)
	g.state = StateAwaitingDiscard(s)
	g.logf("draw %s %s", seatName(s.Player), t)
	return EventNop{}, nil
}

func (g *Game) stepFromAwaitingDiscard(s StateAwaitingDiscard, in Input) (Event, error) {
	switch v := in.(type) {
	case InputDiscard:
		if v.Index < 0 || v.Index >= len(g.hands[s.Player]) {
			return nil, ErrIllegalDiscard
		}
		// Riichi-restricted discard: a seat that has already declared
		// riichi may only discard the just-drawn tile (rightmost slot).
		// The riichi-declaring discard itself is the seat's own choice;
		// every subsequent turn is forced to drawn-only.
		if g.riichiDeclared[s.Player] && !v.Riichi &&
			v.Index != len(g.hands[s.Player])-1 {
			return nil, ErrIllegalDiscard
		}
		// Riichi declaration: validate the four preconditions first.
		// On success we record the deposit + flags before the discard
		// transition completes so doubleRiichi sees the pre-discard
		// noPriorDiscards() / !callsHappened state.
		var declareDoubleRiichi bool
		if v.Riichi {
			if err := g.validateRiichi(s.Player, v.Index); err != nil {
				return nil, err
			}
			declareDoubleRiichi = !g.callsHappened && g.noPriorDiscards()
		}
		t := g.hands[s.Player][v.Index]
		g.hands[s.Player] = append(g.hands[s.Player][:v.Index], g.hands[s.Player][v.Index+1:]...)
		g.discards[s.Player] = append(g.discards[s.Player], t)
		g.hasDiscarded[s.Player] = true
		if s.Player == HumanSeat {
			sortConcealed(g.hands[s.Player])
		}
		if v.Riichi {
			g.scores[s.Player] -= 1000
			g.riichiDeclared[s.Player] = true
			g.ippatsuLive[s.Player] = true
			g.doubleRiichi[s.Player] = declareDoubleRiichi
			g.logf("riichi %s", seatName(s.Player))
		} else if g.riichiDeclared[s.Player] {
			// This is the seat's 2nd-or-later discard since declaring
			// riichi (the declaration discard set v.Riichi=true and
			// took the other branch). Their own non-tsumo turn passed,
			// so ippatsu is no longer reachable for them.
			g.ippatsuLive[s.Player] = false
		}
		g.logf("discard %s %s", seatName(s.Player), t)
		g.state = StateAwaitingClaims{Discard: t, Discarder: s.Player}
		return EventNop{}, nil
	case InputDeclareTsumo:
		concealed := append([]tile.Tile(nil), g.hands[s.Player]...)
		if len(concealed) != 14 {
			return nil, ErrIllegalDiscard
		}
		winning := concealed[len(concealed)-1]
		h := hand.Hand{
			Concealed: concealed,
			Winning:   winning,
			IsTsumo:   true,
			Open:      g.IsHandOpen(s.Player),
		}
		ctx := g.contextForWin(s.Player, true)
		result := calc.Analyze(h, ctx)
		if result == nil {
			// Yakuless win or invalid shape — keep player in current state.
			return nil, ErrYakulessWin
		}
		g.logf("tsumo %s %s", seatName(s.Player), winning)
		g.state = StateRoundOver{Outcome: OutcomeTsumo{
			Winner: s.Player,
			Tile:   winning,
			Hand:   h,
			Result: result,
		}}
		return EventNop{}, nil
	default:
		return nil, ErrInvalidInputForState
	}
}

func (g *Game) stepFromAwaitingClaims(s StateAwaitingClaims, in Input) (Event, error) {
	rc, ok := in.(InputResolveClaims)
	if !ok {
		return nil, ErrInvalidInputForState
	}
	winner, kind, ok := ResolveClaims(rc.Claims, s.Discarder)
	if !ok {
		g.state = StateAwaitingDraw{Player: s.Discarder.Next()}
		return EventNop{}, nil
	}
	switch kind {
	case ClaimRon:
		// Furiten blocks ron for any seat with the machi tile in their
		// own pond. The gate applies universally — humans and bots
		// follow the same permanent-furiten rule. (Temporary furiten
		// across opponent discards is still out of scope; that needs
		// per-seat machi-passed tracking.)
		if g.IsFuriten(winner) {
			return nil, ErrFuritenRon
		}
		concealed := append([]tile.Tile(nil), g.hands[winner]...)
		concealed = append(concealed, s.Discard)
		h := hand.Hand{
			Concealed: concealed,
			Winning:   s.Discard,
			IsTsumo:   false,
			Open:      g.IsHandOpen(winner),
		}
		ctx := g.contextForWin(winner, false)
		result := calc.Analyze(h, ctx)
		if result == nil {
			return nil, ErrYakulessWin
		}
		g.logf("ron %s from %s on %s", seatName(winner), seatName(s.Discarder), s.Discard)
		g.state = StateRoundOver{Outcome: OutcomeRon{
			Winner: winner,
			Loser:  s.Discarder,
			Tile:   s.Discard,
			Hand:   h,
			Result: result,
		}}
		return EventNop{}, nil
	case ClaimPon:
		// Move two copies of s.Discard from claimant's hand into a meld; the
		// discarded tile becomes the third tile of the meld. Then the
		// claimant becomes the active player and must discard.
		if !g.consumeForPon(winner, s.Discard) {
			return nil, fmt.Errorf(
				"game: pon claim from seat %d cannot find two matching tiles",
				winner,
			)
		}
		g.melds[winner] = append(g.melds[winner], Meld{
			Kind:  MeldPon,
			Tiles: []tile.Tile{s.Discard, s.Discard, s.Discard},
			From:  s.Discarder,
		})
		// Pop the discard from discarder's pond — it's been called.
		g.popLastDiscard(s.Discarder)
		g.callsHappened = true
		// Any successful call breaks the ippatsu window for every seat
		// currently in riichi (including the caller themselves, though
		// they can't be in riichi if they're calling pon).
		g.closeAllIppatsuWindows()
		if winner == HumanSeat {
			sortConcealed(g.hands[winner])
		}
		g.logf("pon %s from %s on %s", seatName(winner), seatName(s.Discarder), s.Discard)
		g.state = StateAwaitingDiscard{Player: winner}
		return EventNop{}, nil
	case ClaimChi:
		// Chi support shape: claimant must own two specific tiles named in
		// the Claim. For the v1 minimal path, the resolver places the two
		// tiles + discard into a meld with the resolver-supplied chi tiles.
		c := rc.Claims[winner]
		if !g.consumeForChi(winner, c.ChiTiles[0], c.ChiTiles[1]) {
			return nil, fmt.Errorf(
				"game: chi claim from seat %d cannot find tiles %d+%d",
				winner,
				c.ChiTiles[0],
				c.ChiTiles[1],
			)
		}
		g.melds[winner] = append(g.melds[winner], Meld{
			Kind: MeldChi,
			Tiles: []tile.Tile{
				{ID: c.ChiTiles[0]}, {ID: c.ChiTiles[1]}, s.Discard,
			},
			From: s.Discarder,
		})
		g.popLastDiscard(s.Discarder)
		g.callsHappened = true
		g.closeAllIppatsuWindows()
		if winner == HumanSeat {
			sortConcealed(g.hands[winner])
		}
		g.logf("chi %s from %s on %s", seatName(winner), seatName(s.Discarder), s.Discard)
		g.state = StateAwaitingDiscard{Player: winner}
		return EventNop{}, nil
	}
	return nil, fmt.Errorf("game: unhandled claim kind %d", kind)
}

// closeAllIppatsuWindows clears the ippatsu window for every riichi-declared
// seat. Called from successful pon and chi branches: any call breaks ippatsu
// for everyone currently in riichi.
func (g *Game) closeAllIppatsuWindows() {
	for seat := range Seat(numSeats) {
		if g.riichiDeclared[seat] {
			g.ippatsuLive[seat] = false
		}
	}
}

func (g *Game) consumeForPon(s Seat, t tile.Tile) bool {
	removed := 0
	out := g.hands[s][:0]
	for _, x := range g.hands[s] {
		if removed < 2 && x.ID == t.ID {
			removed++
			continue
		}
		out = append(out, x)
	}
	if removed < 2 {
		return false
	}
	g.hands[s] = out
	return true
}

func (g *Game) consumeForChi(s Seat, a, b uint8) bool {
	hand := g.hands[s]
	idxA, idxB := -1, -1
	for i, x := range hand {
		if idxA < 0 && x.ID == a {
			idxA = i
			continue
		}
		if idxB < 0 && x.ID == b {
			idxB = i
		}
	}
	if idxA < 0 || idxB < 0 {
		return false
	}
	if idxA > idxB {
		idxA, idxB = idxB, idxA
	}
	hand = append(hand[:idxB], hand[idxB+1:]...)
	hand = append(hand[:idxA], hand[idxA+1:]...)
	g.hands[s] = hand
	return true
}

func (g *Game) popLastDiscard(s Seat) {
	d := g.discards[s]
	if len(d) == 0 {
		return
	}
	g.discards[s] = d[:len(d)-1]
}

// contextForWin builds a calc.Context for the given winning seat. Group C
// flags are populated from game state:
//   - Riichi: seat declared riichi this round.
//   - Ippatsu: seat is in riichi and the ippatsu window is still open
//     (closed by any call or by the seat's own next draw).
//   - DoubleRiichi: seat declared riichi on their first uninterrupted intake
//     (no prior discards anywhere, no prior calls).
//   - Haitei: tsumo on the very last live-wall tile.
//   - Houtei: ron on a discard that left the live wall empty.
//   - Tenhou: dealer wins by tsumo with no calls and no prior discards.
//   - Chiihou: non-dealer wins by tsumo with no calls and no prior discards
//     by anyone (including the dealer).
//
// Rinshan / Chankan require kan support, deferred to add-kan-support.
func (g *Game) contextForWin(winner Seat, isTsumo bool) calc.Context {
	ctx := calc.Context{
		SeatWind:     winner.SeatWind(),
		RoundWind:    g.roundWind,
		Dora:         g.doraIndicators,
		Riichi:       g.riichiDeclared[winner],
		Ippatsu:      g.riichiDeclared[winner] && g.ippatsuLive[winner],
		DoubleRiichi: g.doubleRiichi[winner],
	}
	if isTsumo && g.wall.LiveRemaining() == 0 {
		ctx.Haitei = true
	}
	if !isTsumo && g.wall.LiveRemaining() == 0 {
		ctx.Houtei = true
	}
	if isTsumo && !g.callsHappened && g.noPriorDiscards() {
		if winner == SeatEast {
			ctx.Tenhou = true
		} else {
			ctx.Chiihou = true
		}
	}
	return ctx
}

func (g *Game) noPriorDiscards() bool {
	for seat := range Seat(numSeats) {
		if g.hasDiscarded[seat] {
			return false
		}
	}
	return true
}

// tenpaiSeats returns the seats currently in tenpai (shanten == 0 with 13
// concealed tiles). Used for ryuukyoku noten payment bookkeeping.
func (g *Game) tenpaiSeats() []Seat {
	var out []Seat
	for seat := range Seat(numSeats) {
		concealed := g.hands[seat]
		if len(concealed) != 13 {
			continue
		}
		if hand.Shanten(hand.Hand{Concealed: concealed}) == 0 {
			out = append(out, seat)
		}
	}
	return out
}

// testSetHand replaces a seat's hand wholesale (in-package test helper).
func (g *Game) testSetHand(s Seat, tiles []tile.Tile) {
	g.hands[s] = append(g.hands[s][:0], tiles...)
}

// testSetHandTile sets one tile in a seat's hand at the given index
// (in-package test helper).
func (g *Game) testSetHandTile(s Seat, idx int, id uint8) {
	if idx < len(g.hands[s]) {
		g.hands[s][idx] = tile.Tile{ID: id}
	}
}

// testSetState forces the game state directly (in-package test helper).
func (g *Game) testSetState(s State) { g.state = s }

// seatName returns the canonical seat name for log entries.
func seatName(s Seat) string {
	switch s {
	case SeatEast:
		return "East"
	case SeatSouth:
		return "South"
	case SeatWest:
		return "West"
	case SeatNorth:
		return "North"
	}
	return "?"
}

// seatNames maps a slice of seats to their names for log entries.
func seatNames(ss []Seat) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = seatName(s)
	}
	return out
}

// SetTestHand replaces a seat's hand wholesale. Test-only — provided for
// cross-package tests (e.g., internal/play) that need to plant specific
// hands without driving full deal/draw cycles. Production code SHALL NOT
// call this method.
func (g *Game) SetTestHand(s Seat, tiles []tile.Tile) { g.testSetHand(s, tiles) }

// SetTestState forces the game state. Test-only — see SetTestHand.
func (g *Game) SetTestState(s State) { g.testSetState(s) }

// SetTestOpen marks a seat as having an open hand (called melds). Test-only.
// Used to drive the yakuless-win rejection path in play_test.go.
func (g *Game) SetTestOpen(s Seat, open bool) {
	g.testOpen[s] = open
}

// SetTestPond replaces a seat's discard pond wholesale. Test-only — used
// to plant furiten setups (machi tile in own pond) without driving a
// full round of discards.
func (g *Game) SetTestPond(s Seat, tiles []tile.Tile) {
	g.discards[s] = append(g.discards[s][:0], tiles...)
}

// IsHandOpen reports whether the seat's hand has any called melds.
func (g *Game) IsHandOpen(s Seat) bool {
	return g.testOpen[s] || len(g.melds[s]) > 0
}

// Score returns the seat's current point total. Initialized to 25000 in
// New(). Mutated by riichi-deposit deductions and (in a future change) by
// agari payouts and ryuukyoku noten payments.
func (g *Game) Score(s Seat) int { return g.scores[s] }

// IsFuriten reports whether the seat is in permanent furiten — any tile in
// the seat's own pond matches a tile ID in the seat's current machi. Returns
// false for non-tenpai shapes (machi is undefined). v1 only implements
// permanent furiten; temporary furiten across opponent discards lands when
// bot ron is wired in add-smart-ai.
func (g *Game) IsFuriten(s Seat) bool {
	if len(g.hands[s]) != 13 {
		return false
	}
	machi := hand.Machi(hand.Hand{Concealed: g.hands[s]})
	if len(machi) == 0 {
		return false
	}
	machiSet := make(map[uint8]struct{}, len(machi))
	for _, id := range machi {
		machiSet[id] = struct{}{}
	}
	for _, t := range g.discards[s] {
		if _, ok := machiSet[t.ID]; ok {
			return true
		}
	}
	return false
}

// validateRiichi runs the four legality checks for a riichi declaration on
// behalf of seat `s` discarding the tile at `index`. Returns ErrIllegalRiichi
// when any check fails. Does not mutate state.
func (g *Game) validateRiichi(s Seat, index int) error {
	if g.IsHandOpen(s) {
		return ErrIllegalRiichi
	}
	if g.scores[s] < 1000 {
		return ErrIllegalRiichi
	}
	if g.wall.LiveRemaining() < 4 {
		return ErrIllegalRiichi
	}
	// Build the post-discard 13-tile hand WITHOUT mutating state, then
	// check shanten. The drawn-tile invariant means the human's index 13
	// is at len-1, but riichi may also be declared on a non-rightmost
	// tile (the player can sacrifice a sorted-hand tile to enter tenpai).
	postDiscard := make([]tile.Tile, 0, len(g.hands[s])-1)
	postDiscard = append(postDiscard, g.hands[s][:index]...)
	postDiscard = append(postDiscard, g.hands[s][index+1:]...)
	if hand.Shanten(hand.Hand{Concealed: postDiscard}) != 0 {
		return ErrIllegalRiichi
	}
	return nil
}
