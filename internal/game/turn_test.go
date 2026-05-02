package game

import (
	"testing"

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
