package game

import (
	"math/rand/v2"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func newDeterministicBot(seat Seat, seed uint64) *Bot {
	r := rand.New(rand.NewPCG(seed, seed^0xdeadbeef))
	return &Bot{Seat: seat, Rng: r}
}

func TestBotPickDiscardChoosesMostIsolatedTile(t *testing.T) {
	// Hand: 1z + 2m3m4m + 5p6p7p + 1s2s3s + (14th ignored). 1z is the only
	// honor, totally isolated (no neighbors, no copies).
	hand := []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.M5},
		{ID: tile.P8},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.EastWind},
	}
	b := newDeterministicBot(SeatSouth, 1)
	idx := b.PickDiscard(hand)
	if idx < 0 || idx >= len(hand) {
		t.Fatalf("PickDiscard returned out-of-range index %d", idx)
	}
	if hand[idx].ID != tile.EastWind {
		t.Errorf("PickDiscard chose %s, want 1z (the isolated honor)", hand[idx])
	}
}

func TestBotPickDiscardTiebreakByLowestID(t *testing.T) {
	// Hand contains two equally isolated honors: 1z and 7z. Tiebreak picks 1z.
	hand := []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.M5},
		{ID: tile.P8},
		{ID: tile.S5},
		{ID: tile.EastWind},
		{ID: tile.Chun},
	}
	b := newDeterministicBot(SeatSouth, 1)
	idx := b.PickDiscard(hand)
	if hand[idx].ID != tile.EastWind {
		t.Errorf("PickDiscard tiebreak chose %s, want 1z (lowest tile ID)", hand[idx])
	}
}

func TestBotShouldPonYakuhaiAlways(t *testing.T) {
	// Bot at South in round East has two East-wind tiles. East discards East:
	// pon must fire regardless of RNG.
	hand := []tile.Tile{
		{ID: tile.EastWind}, {ID: tile.EastWind}, {ID: tile.M2},
	}
	for seed := uint64(1); seed < 5; seed++ {
		b := newDeterministicBot(SeatSouth, seed)
		// East-wind is yakuhai for South in round East (round wind), and we
		// flag isYakuhai=true to short-circuit the bot's logic.
		if !b.ShouldPon(hand, tile.Tile{ID: tile.EastWind}, true, 5) {
			t.Errorf("ShouldPon(yakuhai=true, hand has 2 east) = false, want true (seed=%d)", seed)
		}
	}
}

func TestBotShouldPonRequiresTwoCopies(t *testing.T) {
	hand := []tile.Tile{
		{ID: tile.M2}, {ID: tile.M3}, {ID: tile.M4},
	}
	b := newDeterministicBot(SeatSouth, 1)
	if b.ShouldPon(hand, tile.Tile{ID: tile.M2}, false, 1) {
		t.Errorf("ShouldPon with only one copy = true, want false")
	}
}

func TestBotShouldPonNonYakuhaiOnlyWhenShantenAtMost2(t *testing.T) {
	hand := []tile.Tile{
		{ID: tile.M5}, {ID: tile.M5}, {ID: tile.P3},
	}
	b := newDeterministicBot(SeatSouth, 1)
	// shanten=5: never pon non-yakuhai, even with 2 copies.
	if b.ShouldPon(hand, tile.Tile{ID: tile.M5}, false, 5) {
		t.Errorf("ShouldPon(non-yakuhai, shanten=5) = true, want false")
	}
}

func TestBotShouldChiFromNonKamichaIsAlwaysFalse(t *testing.T) {
	hand := []tile.Tile{
		{ID: tile.M4}, {ID: tile.M5}, {ID: tile.P1},
	}
	for seed := uint64(1); seed < 10; seed++ {
		b := newDeterministicBot(SeatNorth, seed)
		// East discards 6m. Kamicha-of-North = West, not East → North can't chi.
		if _, ok := b.ShouldChi(hand, tile.Tile{ID: tile.M6}, SeatEast); ok {
			t.Errorf("ShouldChi from non-kamicha = true, want false (seed=%d)", seed)
		}
	}
}

func TestBotShouldChiFromKamichaIsProbabilistic(t *testing.T) {
	// 4m+5m in hand, kamicha discards 6m. South's kamicha is East.
	hand := []tile.Tile{{ID: tile.M4}, {ID: tile.M5}, {ID: tile.P1}}
	disc := tile.Tile{ID: tile.M6}
	calls, total := 0, 200
	for seed := uint64(0); seed < uint64(total); seed++ {
		b := newDeterministicBot(SeatSouth, seed)
		if _, ok := b.ShouldChi(hand, disc, SeatEast); ok {
			calls++
		}
	}
	// 40% probability ± 15% sample noise; over 200 samples this is generous.
	if calls < total/4 || calls > total*55/100 {
		t.Errorf(
			"Chi probability %d/%d = %.2f, want ≈0.40",
			calls,
			total,
			float64(calls)/float64(total),
		)
	}
}

func TestBotShouldKanAlwaysFalseInV1(t *testing.T) {
	b := newDeterministicBot(SeatSouth, 1)
	if b.ShouldKan() {
		t.Errorf("ShouldKan = true, want false (kan deferred to add-kan-support)")
	}
}

func TestBotShouldRiichiLegalOnTenpaiHand(t *testing.T) {
	// 14-tile hand: discarding M5 (index 13) leaves the canonical
	// tenpai shape from tenpaiHandReady() (machi: 4s, 7s).
	hand := append([]tile.Tile(nil), tenpaiHandReady()...)
	hand = append(hand, tile.Tile{ID: tile.M5})

	b := newDeterministicBot(SeatEast, 1)
	declare, idx := b.ShouldRiichi(hand, 25000, 60, false)
	if !declare {
		t.Errorf("ShouldRiichi on tenpai hand returned declare=false, want true")
	}
	if idx < 0 || idx >= len(hand) {
		t.Errorf("ShouldRiichi returned idx=%d, want 0..%d", idx, len(hand)-1)
	}
	// The first tenpai-leaving index is the M5 we appended (index 13)
	// because the underlying 13 tiles form a sorted tenpai shape — removing
	// any of them disrupts the wait.
	if idx != 13 {
		t.Errorf("ShouldRiichi returned idx=%d, want 13 (the appended M5)", idx)
	}
}

func TestBotShouldRiichiRejectedWhenNotTenpai(t *testing.T) {
	hand := []tile.Tile{
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
	b := newDeterministicBot(SeatEast, 1)
	declare, _ := b.ShouldRiichi(hand, 25000, 60, false)
	if declare {
		t.Errorf("ShouldRiichi on non-tenpai hand returned declare=true, want false")
	}
}

func TestBotShouldRiichiRejectedWhenOpen(t *testing.T) {
	hand := append([]tile.Tile(nil), tenpaiHandReady()...)
	hand = append(hand, tile.Tile{ID: tile.M5})

	b := newDeterministicBot(SeatEast, 1)
	declare, _ := b.ShouldRiichi(hand, 25000, 60, true)
	if declare {
		t.Errorf("ShouldRiichi on open hand returned declare=true, want false")
	}
}

func TestBotShouldRiichiRejectedWhenScoreTooLow(t *testing.T) {
	hand := append([]tile.Tile(nil), tenpaiHandReady()...)
	hand = append(hand, tile.Tile{ID: tile.M5})

	b := newDeterministicBot(SeatEast, 1)
	declare, _ := b.ShouldRiichi(hand, 800, 60, false)
	if declare {
		t.Errorf("ShouldRiichi with score=800 returned declare=true, want false")
	}
}

func TestBotShouldRiichiRejectedWhenWallTooLow(t *testing.T) {
	hand := append([]tile.Tile(nil), tenpaiHandReady()...)
	hand = append(hand, tile.Tile{ID: tile.M5})

	b := newDeterministicBot(SeatEast, 1)
	declare, _ := b.ShouldRiichi(hand, 25000, 3, false)
	if declare {
		t.Errorf("ShouldRiichi with wall=3 returned declare=true, want false")
	}
}

func TestBotShouldRiichiRejectedWhenHandSizeWrong(t *testing.T) {
	hand := tenpaiHandReady() // 13 tiles, not 14
	b := newDeterministicBot(SeatEast, 1)
	declare, _ := b.ShouldRiichi(hand, 25000, 60, false)
	if declare {
		t.Errorf("ShouldRiichi on 13-tile hand returned declare=true, want false")
	}
}
