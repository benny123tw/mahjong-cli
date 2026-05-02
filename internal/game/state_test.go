package game

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestNewGameStartsInAwaitingDrawForDealer(t *testing.T) {
	g := New(7)

	st, ok := g.State().(StateAwaitingDraw)
	if !ok {
		t.Fatalf("initial state = %T, want StateAwaitingDraw", g.State())
	}
	if st.Player != SeatEast {
		t.Errorf("initial draw seat = %d, want %d (East)", st.Player, SeatEast)
	}

	for seat := range Seat(numSeats) {
		if got := len(g.Hand(seat)); got != 13 {
			t.Errorf("seat %d hand length = %d, want 13", seat, got)
		}
	}
}

func TestDrawAdvancesAwaitingDrawToAwaitingDiscard(t *testing.T) {
	g := New(7)

	if _, err := g.Step(InputDraw{}); err != nil {
		t.Fatalf("Step(InputDraw) returned err: %v", err)
	}

	st, ok := g.State().(StateAwaitingDiscard)
	if !ok {
		t.Fatalf("state after draw = %T, want StateAwaitingDiscard", g.State())
	}
	if st.Player != SeatEast {
		t.Errorf("discard seat after East draws = %d, want %d (East)", st.Player, SeatEast)
	}
	if got := len(g.Hand(SeatEast)); got != 14 {
		t.Errorf("East hand after draw = %d, want 14", got)
	}
}

func TestDiscardAdvancesAwaitingDiscardToAwaitingClaims(t *testing.T) {
	g := New(7)
	mustStep(t, g, InputDraw{})

	if _, err := g.Step(InputDiscard{Index: 0}); err != nil {
		t.Fatalf("Step(InputDiscard) returned err: %v", err)
	}

	st, ok := g.State().(StateAwaitingClaims)
	if !ok {
		t.Fatalf("state after discard = %T, want StateAwaitingClaims", g.State())
	}
	if st.Discarder != SeatEast {
		t.Errorf("discarder = %d, want %d (East)", st.Discarder, SeatEast)
	}
	if got := len(g.Hand(SeatEast)); got != 13 {
		t.Errorf("East hand after discard = %d, want 13", got)
	}
	if got := len(g.Discards(SeatEast)); got != 1 {
		t.Errorf("East discards after discard = %d, want 1", got)
	}
}

func TestNoClaimAdvancesAwaitingClaimsToNextPlayersAwaitingDraw(t *testing.T) {
	g := New(7)
	mustStep(t, g, InputDraw{})
	mustStep(t, g, InputDiscard{Index: 0})

	mustStep(t, g, InputResolveClaims{Claims: nil})

	st, ok := g.State().(StateAwaitingDraw)
	if !ok {
		t.Fatalf("state after no-claim resolve = %T, want StateAwaitingDraw", g.State())
	}
	if st.Player != SeatSouth {
		t.Errorf("next draw seat after East discards = %d, want %d (South)", st.Player, SeatSouth)
	}
}

func TestPonClaimAdvancesAwaitingClaimsToClaimantsAwaitingDiscard(t *testing.T) {
	// Synthesize a hand setup: South has two copies of East's discard.
	g := New(7)
	mustStep(t, g, InputDraw{})
	// Force East's discard to be a tile we know South has two of.
	plant := plantTwoCopies(t, g, SeatSouth)
	mustStepDiscardTile(t, g, plant)

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		SeatSouth: {Kind: ClaimPon},
	}})
	if err != nil {
		t.Fatalf("Step(InputResolveClaims pon) returned err: %v", err)
	}

	st, ok := g.State().(StateAwaitingDiscard)
	if !ok {
		t.Fatalf("state after pon = %T, want StateAwaitingDiscard", g.State())
	}
	if st.Player != SeatSouth {
		t.Errorf("discard seat after South's pon = %d, want %d (South)", st.Player, SeatSouth)
	}
}

func TestExhaustedLiveWallTransitionsToRoundOverRyuukyoku(t *testing.T) {
	g := New(7)

	// Drive 70 normal draw/discard turns to exhaust the live wall.
	for range 70 {
		mustStep(t, g, InputDraw{})
		mustStep(t, g, InputDiscard{Index: 0})
		mustStep(t, g, InputResolveClaims{Claims: nil})
	}

	// The 71st draw attempt should land in RoundOver.
	if _, err := g.Step(InputDraw{}); err != nil {
		t.Fatalf("Step(InputDraw) on exhausted wall returned err: %v", err)
	}
	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after wall exhausts = %T, want StateRoundOver", g.State())
	}
	if _, ok := st.Outcome.(OutcomeRyuukyoku); !ok {
		t.Errorf("outcome on exhausted wall = %T, want OutcomeRyuukyoku", st.Outcome)
	}
}

func TestRonClaimTransitionsToRoundOver(t *testing.T) {
	// Synthesize: South is tenpai with 1z pair as winning shape, ron on East's
	// discard of 1z.
	g := newGameWithRonReady(t, 7)

	st, ok := g.State().(StateAwaitingClaims)
	if !ok {
		t.Fatalf("setup state = %T, want StateAwaitingClaims", g.State())
	}
	_ = st

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		SeatSouth: {Kind: ClaimRon},
	}})
	if err != nil {
		t.Fatalf("Step(InputResolveClaims ron) returned err: %v", err)
	}
	final, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after ron = %T, want StateRoundOver", g.State())
	}
	out, ok := final.Outcome.(OutcomeRon)
	if !ok {
		t.Fatalf("outcome after ron = %T, want OutcomeRon", final.Outcome)
	}
	if out.Winner != SeatSouth {
		t.Errorf("ron winner = %d, want %d (South)", out.Winner, SeatSouth)
	}
	if out.Loser != SeatEast {
		t.Errorf("ron loser = %d, want %d (East)", out.Loser, SeatEast)
	}
}

// mustStep calls Step and fails the test on error.
func mustStep(t *testing.T, g *Game, in Input) {
	t.Helper()
	if _, err := g.Step(in); err != nil {
		t.Fatalf("Step(%T) failed: %v", in, err)
	}
}

// mustStepDiscardTile finds the given tile ID in the active player's hand and
// discards it. Useful when the test needs East to discard a specific tile.
func mustStepDiscardTile(t *testing.T, g *Game, id uint8) {
	t.Helper()
	st, ok := g.State().(StateAwaitingDiscard)
	if !ok {
		t.Fatalf("mustStepDiscardTile: state is %T, want StateAwaitingDiscard", g.State())
	}
	hand := g.Hand(st.Player)
	for i, x := range hand {
		if x.ID == id {
			mustStep(t, g, InputDiscard{Index: i})
			return
		}
	}
	t.Fatalf("mustStepDiscardTile: tile id %d not found in seat %d hand", id, st.Player)
}

// plantTwoCopies forces the active discarder's hand to contain a tile that
// `target` has at least two copies of, then returns that tile id. The active
// discarder MUST already have drawn (state is AwaitingDiscard). Test helper.
func plantTwoCopies(t *testing.T, g *Game, target Seat) uint8 {
	t.Helper()
	targetHand := g.Hand(target)
	var counts [34]int
	for _, x := range targetHand {
		counts[x.ID]++
	}
	for id, c := range counts {
		if c >= 2 {
			st, ok := g.State().(StateAwaitingDiscard)
			if !ok {
				t.Fatalf("plantTwoCopies: state is %T, want StateAwaitingDiscard", g.State())
			}
			g.testSetHandTile(st.Player, 0, uint8(id))
			return uint8(id)
		}
	}
	t.Fatalf("plantTwoCopies: target seat %d has no tile with two copies", target)
	return 0
}

func TestRyuukyokuEnumeratesTenpaiSeats(t *testing.T) {
	g := New(11)
	for range 70 {
		mustStep(t, g, InputDraw{})
		mustStep(t, g, InputDiscard{Index: 0})
		mustStep(t, g, InputResolveClaims{Claims: nil})
	}
	// Plant South into a known tenpai shape just before the exhaust draw, so
	// the ryuukyoku enumeration MUST report South. The other seats' hands are
	// whatever the random play left them in — they may or may not be tenpai;
	// the assertion is only about South's inclusion.
	g.testSetHand(SeatSouth, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M7},
		{ID: tile.EastWind},
	})

	mustStep(t, g, InputDraw{})
	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after wall exhausts = %T, want StateRoundOver", g.State())
	}
	out, ok := st.Outcome.(OutcomeRyuukyoku)
	if !ok {
		t.Fatalf("outcome = %T, want OutcomeRyuukyoku", st.Outcome)
	}
	foundSouth := false
	for _, s := range out.TenpaiPlayers {
		if s == SeatSouth {
			foundSouth = true
		}
	}
	if !foundSouth {
		t.Errorf(
			"OutcomeRyuukyoku.TenpaiPlayers = %v, want to contain South (seat %d)",
			out.TenpaiPlayers,
			SeatSouth,
		)
	}
}

func TestTsumoTransitionsToRoundOver(t *testing.T) {
	g := New(13)
	// South: chiitoitsu winning shape (7 pairs of m).
	g.testSetHand(SeatSouth, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.M9},
	})
	g.testSetState(StateAwaitingDiscard{Player: SeatSouth})

	if _, err := g.Step(InputDeclareTsumo{}); err != nil {
		t.Fatalf("Step(InputDeclareTsumo) returned err: %v", err)
	}
	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after tsumo = %T, want StateRoundOver", g.State())
	}
	out, ok := st.Outcome.(OutcomeTsumo)
	if !ok {
		t.Fatalf("outcome = %T, want OutcomeTsumo", st.Outcome)
	}
	if out.Winner != SeatSouth {
		t.Errorf("tsumo winner = %d, want %d (South)", out.Winner, SeatSouth)
	}
}

// newGameWithRonReady builds a Game where South is tenpai on East's wind
// (East round + South seat) yakuhai-pair pattern and East is about to discard
// East-wind. Used to test the ron-claim transition.
func newGameWithRonReady(t *testing.T, seed int64) *Game {
	t.Helper()
	g := New(seed)

	// South's hand: a tenpai shape that wins on East-wind via shanpon.
	// 2m2m3m3m4m4m5m5m6m6m7m7m1z — winning on 1z makes seven pairs (chiitoitsu)
	// with East-wind as the seventh pair. Chiitoitsu is itself a yaku, so the
	// win is yaku-bearing.
	southHand := []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M7},
		{ID: tile.EastWind},
	}
	g.testSetHand(SeatSouth, southHand)

	// Force East's draw and discard so the discard is East-wind.
	mustStep(t, g, InputDraw{})
	st, ok := g.State().(StateAwaitingDiscard)
	if !ok {
		t.Fatalf("newGameWithRonReady: state after East draw = %T", g.State())
	}
	// Plant East-wind in East's hand at index 0 and discard it.
	g.testSetHandTile(st.Player, 0, tile.EastWind)
	mustStep(t, g, InputDiscard{Index: 0})
	return g
}
