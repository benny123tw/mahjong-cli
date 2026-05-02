package game

import (
	"errors"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestIsFuritenWhenMachiTileInOwnPond(t *testing.T) {
	g := New(7)
	// Plant tenpai with machi {4s, 7s} (ryanmen on 5s+6s).
	g.testSetHand(HumanSeat, tenpaiHandReady())
	// Plant 4s in own pond — furiten.
	g.discards[HumanSeat] = append(g.discards[HumanSeat], tile.Tile{ID: tile.S4})

	if !g.IsFuriten(HumanSeat) {
		t.Errorf("IsFuriten(Human) = false with machi tile in own pond, want true")
	}
}

func TestIsFuritenFalseWhenMachiTilesAbsentFromOwnPond(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, tenpaiHandReady())
	// Pond contains tiles that are NOT in machi {4s, 7s}.
	g.discards[HumanSeat] = []tile.Tile{
		{ID: tile.M9}, {ID: tile.P5}, {ID: tile.EastWind},
	}

	if g.IsFuriten(HumanSeat) {
		t.Errorf("IsFuriten(Human) = true with no machi tile in pond, want false")
	}
}

func TestIsFuritenFalseOnNonTenpaiHand(t *testing.T) {
	g := New(7)
	// Garbage 13-tile hand with shanten > 0.
	g.testSetHand(HumanSeat, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M3},
		{ID: tile.M5},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.P1},
		{ID: tile.P3},
		{ID: tile.P5},
		{ID: tile.P7},
		{ID: tile.P9},
		{ID: tile.S1},
		{ID: tile.S3},
		{ID: tile.S5},
	})
	g.discards[HumanSeat] = []tile.Tile{{ID: tile.M1}}

	if g.IsFuriten(HumanSeat) {
		t.Errorf("IsFuriten on non-tenpai hand = true, want false (machi is undefined)")
	}
}

func TestHumanRonOnYakuBearingDiscard(t *testing.T) {
	g := New(7)
	// Tenpai: 234m+234p+234s+5s6s+chun-chun, ryanmen on 4s/7s.
	// Add chun-pair-as-yakuhai for yaku, and discarder-supplied 7s for win.
	// Actually let's go simpler: tanyao tenpai shape so yaku is automatic.
	// Wait, ron on East's discard from open hand vs concealed — must be concealed for menzen-tsumo, but not for tanyao.
	// Use tanyao: all simples, ron on a simple. tenpai 234m+234p+234s+22s+5s6s7s would already win — we need a wait.
	// Let's do: 234m+234p+234s+5s6s+44m, ron on 7s: 5s6s+7s makes a run, plus 44m pair, all simples → tanyao. But concealed (no melds) so menzen too.
	g.testSetHand(HumanSeat, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.S5},
		{ID: tile.S6},
	})
	g.testSetState(StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: SeatEast})

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		HumanSeat: {Kind: ClaimRon},
	}})
	if err != nil {
		t.Fatalf("Step(ClaimRon) returned err: %v", err)
	}
	st, ok := g.state.(StateRoundOver)
	if !ok {
		t.Fatalf("state after ron = %T, want StateRoundOver", g.state)
	}
	out, ok := st.Outcome.(OutcomeRon)
	if !ok {
		t.Fatalf("outcome after ron = %T, want OutcomeRon", st.Outcome)
	}
	if out.Winner != HumanSeat {
		t.Errorf("ron winner = %d, want HumanSeat", out.Winner)
	}
	if out.Loser != SeatEast {
		t.Errorf("ron loser = %d, want SeatEast", out.Loser)
	}
}

func TestHumanRonRejectedWhenFuriten(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.S5},
		{ID: tile.S6},
	})
	// Plant the machi 7s in human's own pond → permanent furiten.
	g.discards[HumanSeat] = append(g.discards[HumanSeat], tile.Tile{ID: tile.S7})
	g.testSetState(StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: SeatEast})

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		HumanSeat: {Kind: ClaimRon},
	}})
	if !errors.Is(err, ErrFuritenRon) {
		t.Errorf("furiten ron returned err=%v, want ErrFuritenRon", err)
	}
	if _, ok := g.state.(StateAwaitingClaims); !ok {
		t.Errorf("state after furiten ron = %T, want StateAwaitingClaims (unchanged)", g.state)
	}
}
