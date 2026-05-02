package game

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestSeatWindForDealerRelative(t *testing.T) {
	// East 1: dealer is SeatEast. Seat winds match seat IDs (East→East, South→South, ...).
	g1 := NewWithDealer(7, SeatEast, tile.EastWind)
	cases1 := []struct {
		seat Seat
		want uint8
	}{
		{SeatEast, tile.EastWind},
		{SeatSouth, tile.SouthWind},
		{SeatWest, tile.WestWind},
		{SeatNorth, tile.NorthWind},
	}
	for _, c := range cases1 {
		if got := g1.SeatWindFor(c.seat); got != c.want {
			t.Errorf("E1: SeatWindFor(seat=%d) = %d, want %d", c.seat, got, c.want)
		}
	}

	// East 2: dealer is SeatSouth. Dealer is East-wind; SeatEast wraps to North-wind.
	g2 := NewWithDealer(7, SeatSouth, tile.EastWind)
	cases2 := []struct {
		seat Seat
		want uint8
	}{
		{SeatSouth, tile.EastWind},
		{SeatWest, tile.SouthWind},
		{SeatNorth, tile.WestWind},
		{SeatEast, tile.NorthWind},
	}
	for _, c := range cases2 {
		if got := g2.SeatWindFor(c.seat); got != c.want {
			t.Errorf("E2: SeatWindFor(seat=%d) = %d, want %d", c.seat, got, c.want)
		}
	}
}

func TestContextForWinUsesSeatWindFor(t *testing.T) {
	// East 2 dealer = SeatSouth. SeatNorth's hand-relative wind is West.
	g := NewWithDealer(7, SeatSouth, tile.EastWind)
	ctx := g.contextForWin(SeatNorth, true)
	if ctx.SeatWind != tile.WestWind {
		t.Errorf(
			"contextForWin(North).SeatWind = %d, want WestWind=%d (North is West-wind when dealer is South)",
			ctx.SeatWind,
			tile.WestWind,
		)
	}
	if ctx.RoundWind != tile.EastWind {
		t.Errorf("contextForWin(North).RoundWind = %d, want EastWind", ctx.RoundWind)
	}
}

func TestNewWithDealerStartsDealerToDraw(t *testing.T) {
	g := NewWithDealer(7, SeatSouth, tile.EastWind)
	st, ok := g.State().(StateAwaitingDraw)
	if !ok {
		t.Fatalf("initial state = %T, want StateAwaitingDraw", g.State())
	}
	if st.Player != SeatSouth {
		t.Errorf("initial draw player = %d, want SeatSouth (the dealer)", st.Player)
	}
}

func TestLegacyNewDelegatesToNewWithDealer(t *testing.T) {
	// Backwards-compat invariant: New(seed) must produce an East-dealer hand
	// with East-wind round, and SeatWindFor(SeatEast) == EastWind.
	g := New(7)
	if got := g.SeatWindFor(SeatEast); got != tile.EastWind {
		t.Errorf("New(7).SeatWindFor(East) = %d, want EastWind", got)
	}
	if g.RoundWind() != tile.EastWind {
		t.Errorf("New(7).RoundWind() = %d, want EastWind", g.RoundWind())
	}
	st, ok := g.State().(StateAwaitingDraw)
	if !ok || st.Player != SeatEast {
		t.Errorf("New(7) initial state = %v, want AwaitingDraw{East}", g.State())
	}
}

func TestContextForWinPopulatesCalledMelds(t *testing.T) {
	g := New(7)
	// Concealed: 234m + 234p + 234s + 2m + 2m (11 tiles, pair-completing
	// rinshan replacement at the end). Open ankan on 5p contributes 3 tiles
	// to effectiveConcealed; open pon on 7s contributes 3 tiles. The 14-tile
	// shape (11 concealed + 3 ankan) doesn't include the pon yet — we need
	// 11 concealed + 3 ankan + 3 pon = 17. That overshoots, so use a 8-tile
	// concealed hand: 234m + 234p + 2m + 2m (8 tiles), plus ankan(5p)=3 and
	// pon(7s)=3 totals 14. Sets: 2m3m4m, 2p3p4p, 5p5p5p ankan, 7s7s7s pon,
	// pair 2m2m. Winning tile is the second 2m (last in slice).
	g.testSetHand(SeatEast, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.M2},
		{ID: tile.M2},
	})
	g.melds[SeatEast] = []Meld{
		{
			Kind:    MeldKan,
			KanKind: KanAnkan,
			Tiles:   []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		},
		{
			Kind:  MeldPon,
			Tiles: []tile.Tile{{ID: tile.S7}, {ID: tile.S7}, {ID: tile.S7}},
			From:  SeatSouth,
		},
	}
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})

	_, err := g.Step(InputDeclareTsumo{})
	if err != nil {
		t.Fatalf("Step(tsumo) returned err: %v", err)
	}
	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after tsumo = %T, want StateRoundOver", g.State())
	}
	out, ok := st.Outcome.(OutcomeTsumo)
	if !ok {
		t.Fatalf("outcome after tsumo = %T, want OutcomeTsumo", st.Outcome)
	}

	if got := len(out.Hand.CalledMelds); got != 2 {
		t.Fatalf("Hand.CalledMelds len = %d, want 2", got)
	}
	want := map[hand.CalledKind]uint8{
		hand.CalledAnkan: tile.P5,
		hand.CalledPon:   tile.S7,
	}
	for _, cm := range out.Hand.CalledMelds {
		baseID, ok := want[cm.Kind]
		if !ok {
			t.Errorf("unexpected CalledMeld.Kind = %d", cm.Kind)
			continue
		}
		if cm.BaseID != baseID {
			t.Errorf("CalledMeld{Kind=%d}.BaseID = %d, want %d", cm.Kind, cm.BaseID, baseID)
		}
	}
}
