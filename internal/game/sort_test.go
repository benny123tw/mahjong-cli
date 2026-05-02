package game

import (
	"slices"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// isSortedByID reports whether tiles are in non-decreasing tile-ID order.
func isSortedByID(tiles []tile.Tile) bool {
	for i := 1; i < len(tiles); i++ {
		if tiles[i].ID < tiles[i-1].ID {
			return false
		}
	}
	return true
}

func tileIDs(tiles []tile.Tile) []uint8 {
	out := make([]uint8, len(tiles))
	for i, t := range tiles {
		out[i] = t.ID
	}
	return out
}

// TestHumanHandSortedAfterDeal asserts that the human seat's 13-tile hand is
// in canonical (ascending tile-ID) order immediately after New() deals.
func TestHumanHandSortedAfterDeal(t *testing.T) {
	g := New(42)
	south := g.Hand(HumanSeat)
	if len(south) != 13 {
		t.Fatalf("South hand after deal = %d tiles, want 13", len(south))
	}
	if !isSortedByID(south) {
		t.Errorf("South hand after deal is not sorted by ID: %v", tileIDs(south))
	}
}

// TestDrawAppendsUnsortedAt14 asserts that the just-drawn 14th tile is
// preserved at index 13 (rightmost slot) regardless of where it would fall
// in canonical order, so the player can identify what they drew.
func TestDrawAppendsUnsortedAt14(t *testing.T) {
	g := New(42)
	// Plant a known sorted 13-tile hand whose last tile has a high ID,
	// then plant a low-ID tile at the top of the wall so the next draw
	// definitely sorts to a non-rightmost canonical position.
	g.testSetHand(HumanSeat, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.Haku},
		{ID: tile.Haku},
		{ID: tile.Haku},
	})
	g.testSetState(StateAwaitingDraw{Player: HumanSeat})

	mustStep(t, g, InputDraw{})

	hand := g.Hand(HumanSeat)
	if len(hand) != 14 {
		t.Fatalf("hand after draw = %d tiles, want 14", len(hand))
	}
	// Indices 0..12 should remain in the originally-planted sorted order.
	wantPrefix := []uint8{
		tile.M2, tile.M3, tile.M4,
		tile.P5, tile.P5, tile.P6, tile.P7,
		tile.S2, tile.S3, tile.S4,
		tile.Haku, tile.Haku, tile.Haku,
	}
	gotPrefix := tileIDs(hand[:13])
	if !slices.Equal(gotPrefix, wantPrefix) {
		t.Errorf(
			"hand[0..12] after draw = %v, want sorted prefix %v",
			gotPrefix,
			wantPrefix,
		)
	}
}

// TestDiscardDrawnTileLeavesSortedHand asserts that discarding the just-drawn
// 14th tile produces a sorted 13-tile hand (the trivial case — sorted prefix
// is left untouched).
func TestDiscardDrawnTileLeavesSortedHand(t *testing.T) {
	g := New(42)
	mustStepHumanThroughBotsToDraw(t, g)

	hand := g.Hand(HumanSeat)
	if len(hand) != 14 {
		t.Fatalf("hand before discard = %d, want 14", len(hand))
	}
	// Discard index 13 (the drawn tile).
	mustStep(t, g, InputDiscard{Index: 13})

	hand = g.Hand(HumanSeat)
	if len(hand) != 13 {
		t.Fatalf("hand after discarding drawn tile = %d, want 13", len(hand))
	}
	if !isSortedByID(hand) {
		t.Errorf("hand after discarding drawn tile is not sorted: %v", tileIDs(hand))
	}
}

// TestDiscardSortedHandTileResortsRemaining asserts that discarding a tile
// from indices 0..12 (a sorted-hand tile, not the drawn tile) leaves the
// drawn tile slotted into canonical position in the resulting 13-tile hand.
func TestDiscardSortedHandTileResortsRemaining(t *testing.T) {
	g := New(42)
	// Plant a deterministic 13-tile sorted hand and a known drawn tile.
	g.testSetHand(HumanSeat, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S1},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.Chun},
	})
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})
	// Manually append drawn M4 at index 13 to mimic post-draw state.
	g.hands[HumanSeat] = append(g.hands[HumanSeat], tile.Tile{ID: tile.M4})

	// Discard S1 at index 8 (a sorted-hand tile).
	mustStep(t, g, InputDiscard{Index: 8})

	hand := g.Hand(HumanSeat)
	if len(hand) != 13 {
		t.Fatalf("hand after discard = %d, want 13", len(hand))
	}
	want := []uint8{
		tile.M1, tile.M1, tile.M2, tile.M3, tile.M4,
		tile.P5, tile.P5, tile.P6, tile.P7,
		tile.S1,
		tile.Chun, tile.Chun, tile.Chun,
	}
	if got := tileIDs(hand); !slices.Equal(got, want) {
		t.Errorf("hand after discarding sorted-hand tile = %v, want %v", got, want)
	}
}

// TestBotHandsAreNotSortedAfterDeal asserts that bot seats' hands are NOT
// auto-sorted by the engine. With seed=42 at least one of the three bots
// must have an unsorted hand — the probability of all three being sorted by
// chance is astronomically small.
func TestBotHandsAreNotSortedAfterDeal(t *testing.T) {
	g := New(42)
	for _, bot := range []Seat{SeatEast, SeatWest, SeatNorth} {
		if !isSortedByID(g.Hand(bot)) {
			return // found at least one unsorted bot hand — invariant holds
		}
	}
	t.Errorf(
		"all three bot hands were sorted with seed=42; engine appears to be sorting bots' hands (it should only sort the human's)",
	)
}

// mustStepHumanThroughBotsToDraw cycles the East/West/North bots through one
// draw+discard each so the human (South) becomes the active drawer, then
// fires InputDraw so the human enters AwaitingDiscard with 14 tiles.
func mustStepHumanThroughBotsToDraw(t *testing.T, g *Game) {
	t.Helper()
	// East draws + discards
	mustStep(t, g, InputDraw{})
	mustStep(t, g, InputDiscard{Index: 0})
	mustStep(t, g, InputResolveClaims{Claims: nil})
	// South draws (human's first draw)
	mustStep(t, g, InputDraw{})
}
