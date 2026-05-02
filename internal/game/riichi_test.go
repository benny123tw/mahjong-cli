package game

import (
	"errors"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// tenpaiHandReady is a deterministic 13-tile hand that is at shanten=0
// (tenpai) when planted directly, before any discard. Used as the post-
// discard fixture.
func tenpaiHandReady() []tile.Tile {
	return []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S6}, // ryanmen waiting on 4s/7s
		{ID: tile.Haku},
		{ID: tile.Haku},
	}
}

// fourteenTileTenpaiHand is a 14-tile hand where discarding index 13 leaves
// the tenpaiHandReady() shape exactly.
func fourteenTileTenpaiHand() []tile.Tile {
	out := append([]tile.Tile(nil), tenpaiHandReady()...)
	return append(out, tile.Tile{ID: tile.M5}) // unrelated tile to discard
}

func TestRiichiDeclarationSucceedsOnTenpaiConcealedHand(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	_, err := g.Step(InputDiscard{Index: 13, Riichi: true})
	if err != nil {
		t.Fatalf("Step(riichi-declare) returned err: %v", err)
	}
	if !g.riichiDeclared[HumanSeat] {
		t.Errorf("riichiDeclared[Human] = false after successful declaration, want true")
	}
	if g.scores[HumanSeat] != 24000 {
		t.Errorf(
			"scores[Human] after riichi = %d, want 24000 (1000 deposit deducted)",
			g.scores[HumanSeat],
		)
	}
	if !g.ippatsuLive[HumanSeat] {
		t.Errorf("ippatsuLive[Human] = false after declaration, want true")
	}
}

func TestRiichiRejectedWhenHandIsOpen(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})
	g.SetTestOpen(HumanSeat, true) // simulate prior pon

	_, err := g.Step(InputDiscard{Index: 13, Riichi: true})
	if !errors.Is(err, ErrIllegalRiichi) {
		t.Errorf("Step(riichi) on open hand returned err=%v, want ErrIllegalRiichi", err)
	}
	if g.riichiDeclared[HumanSeat] {
		t.Errorf("riichiDeclared[Human] = true after rejected riichi, want false")
	}
}

func TestRiichiRejectedWhenWallTooSmall(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})
	// Drain wall to 3 tiles by repeatedly drawing.
	for g.wall.LiveRemaining() > 3 {
		_, _ = g.wall.Draw()
	}

	_, err := g.Step(InputDiscard{Index: 13, Riichi: true})
	if !errors.Is(err, ErrIllegalRiichi) {
		t.Errorf("Step(riichi) with wall<4 returned err=%v, want ErrIllegalRiichi", err)
	}
}

func TestRiichiRejectedWhenPostDiscardNotTenpai(t *testing.T) {
	g := New(7)
	// 14 tiles, but discarding index 0 leaves a non-tenpai hand.
	noisy := []tile.Tile{
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
		{ID: tile.S7},
	}
	g.testSetHand(HumanSeat, noisy)
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	_, err := g.Step(InputDiscard{Index: 0, Riichi: true})
	if !errors.Is(err, ErrIllegalRiichi) {
		t.Errorf(
			"Step(riichi) on non-tenpai-after-discard returned err=%v, want ErrIllegalRiichi",
			err,
		)
	}
}

func TestRiichiRejectedWhenInsufficientScore(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})
	g.scores[HumanSeat] = 800

	_, err := g.Step(InputDiscard{Index: 13, Riichi: true})
	if !errors.Is(err, ErrIllegalRiichi) {
		t.Errorf("Step(riichi) with score<1000 returned err=%v, want ErrIllegalRiichi", err)
	}
}

func TestRiichiRestrictedDiscardEnforced(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	// Declare riichi on this turn.
	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("riichi declaration failed: %v", err)
	}

	// Pass through claims, opponents' draws/discards until human's next turn.
	// Simplest: directly set up another AwaitingDiscard with a 14-tile hand.
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	// Try a non-rightmost discard — should fail.
	_, err := g.Step(InputDiscard{Index: 0})
	if !errors.Is(err, ErrIllegalDiscard) {
		t.Errorf(
			"riichi-declared seat discarding non-drawn tile returned err=%v, want ErrIllegalDiscard",
			err,
		)
	}

	// Discarding the rightmost tile (index 13) succeeds.
	if _, err := g.Step(InputDiscard{Index: 13}); err != nil {
		t.Errorf("riichi-declared seat discarding drawn tile returned err=%v, want nil", err)
	}
}

func TestDoubleRiichiOnFirstUninterruptedDraw(t *testing.T) {
	g := New(7)
	// East's first turn — set up dealer-tenpai-on-first-draw scenario.
	// Plant East at 14-tile tenpai BEFORE any discard or call has happened.
	tenpai := append([]tile.Tile(nil), tenpaiHandReady()...)
	tenpai = append(tenpai, tile.Tile{ID: tile.M5})
	g.testSetHand(SeatEast, tenpai)
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})

	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("dealer riichi declaration failed: %v", err)
	}
	if !g.doubleRiichi[SeatEast] {
		t.Errorf("doubleRiichi[East] = false after first-uninterrupted riichi, want true")
	}
}

func TestIppatsuOpenAfterRiichiDeclaration(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("riichi declaration failed: %v", err)
	}
	if !g.ippatsuLive[HumanSeat] {
		t.Errorf("ippatsuLive[Human] = false right after declaration, want true")
	}
}

func TestIppatsuClosesOnHumansSecondDiscard(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("riichi declaration failed: %v", err)
	}
	if !g.ippatsuLive[HumanSeat] {
		t.Fatalf("ippatsuLive[Human] = false after declaration, want true")
	}

	// Re-plant a 14-tile hand (simulating opponents played + human drew)
	// and force a non-riichi discard from the human.
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})
	if _, err := g.Step(InputDiscard{Index: 13}); err != nil {
		t.Fatalf("post-riichi discard failed: %v", err)
	}
	if g.ippatsuLive[HumanSeat] {
		t.Errorf("ippatsuLive[Human] = true after 2nd-since-declaration discard, want false")
	}
}

func TestIppatsuClosesOnAnyCall(t *testing.T) {
	g := New(7)
	g.testSetHand(HumanSeat, fourteenTileTenpaiHand())
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("riichi declaration failed: %v", err)
	}
	if !g.ippatsuLive[HumanSeat] {
		t.Fatalf("ippatsuLive[Human] = false after declaration, want true")
	}

	// Set up a separate AwaitingClaims with West holding a pair to pon East's discard.
	g.testSetHand(SeatWest, []tile.Tile{
		{ID: tile.M3},
		{ID: tile.M3},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M8},
		{ID: tile.M9},
	})
	g.testSetState(
		StateAwaitingClaims{Discard: tile.Tile{ID: tile.M3}, Discarder: SeatEast},
	)
	g.discards[SeatEast] = append(g.discards[SeatEast], tile.Tile{ID: tile.M3})

	if _, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		SeatWest: {Kind: ClaimPon},
	}}); err != nil {
		t.Fatalf("West pon failed: %v", err)
	}
	if g.ippatsuLive[HumanSeat] {
		t.Errorf("ippatsuLive[Human] = true after a call broke the round, want false")
	}
}

func TestDoubleRiichiNotSetAfterPriorDiscards(t *testing.T) {
	g := New(7)
	// Drive East through one normal discard cycle first.
	mustStep(t, g, InputDraw{})
	mustStep(t, g, InputDiscard{Index: 0})
	mustStep(t, g, InputResolveClaims{Claims: nil})
	// Now South is up. Plant tenpai for riichi declaration.
	tenpai := append([]tile.Tile(nil), tenpaiHandReady()...)
	tenpai = append(tenpai, tile.Tile{ID: tile.M5})
	g.testSetHand(HumanSeat, tenpai)
	g.testSetState(StateAwaitingDiscard{Player: HumanSeat})

	if _, err := g.Step(InputDiscard{Index: 13, Riichi: true}); err != nil {
		t.Fatalf("south riichi declaration failed: %v", err)
	}
	if g.doubleRiichi[HumanSeat] {
		t.Errorf("doubleRiichi[Human] = true after a prior East discard, want false")
	}
	if !g.riichiDeclared[HumanSeat] {
		t.Errorf("riichiDeclared[Human] = false, want true (regular riichi should still succeed)")
	}
}
