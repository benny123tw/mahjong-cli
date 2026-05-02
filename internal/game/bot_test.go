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

func TestBotGenbutsu(t *testing.T) {
	pond := []tile.Tile{
		{ID: tile.M2}, {ID: tile.P5}, {ID: tile.S7}, {ID: tile.EastWind},
	}
	if !Genbutsu(pond, tile.Tile{ID: tile.P5}) {
		t.Errorf("Genbutsu(pond, 5p) = false, want true (5p in pond)")
	}
	if !Genbutsu(pond, tile.Tile{ID: tile.EastWind}) {
		t.Errorf("Genbutsu(pond, EastWind) = false, want true (East in pond)")
	}
	if Genbutsu(pond, tile.Tile{ID: tile.P3}) {
		t.Errorf("Genbutsu(pond, 3p) = true, want false (3p not in pond)")
	}
	if Genbutsu(nil, tile.Tile{ID: tile.M5}) {
		t.Errorf("Genbutsu(nil, 5m) = true, want false (empty pond)")
	}
	if Genbutsu([]tile.Tile{}, tile.Tile{ID: tile.M5}) {
		t.Errorf("Genbutsu(empty, 5m) = true, want false")
	}
}

func TestBotSujiSafe(t *testing.T) {
	// Each entry: (pond rank, safe rank that the pond rank covers).
	// Per the suji theory: 4-in-pond covers 1+7, 5-in-pond covers 2+8,
	// 6-in-pond covers 3+9.
	cases := []struct {
		name      string
		pondID    uint8
		safeID    uint8
		wantSafe  bool
		notSafeID uint8 // tile that should NOT be suji-safe given this pond
	}{
		{"4m covers 1m", tile.M4, tile.M1, true, tile.M5},
		{"4m covers 7m", tile.M4, tile.M7, true, tile.M5},
		{"5p covers 2p", tile.P5, tile.P2, true, tile.P5},
		{"5p covers 8p", tile.P5, tile.P8, true, tile.P5},
		{"6s covers 3s", tile.S6, tile.S3, true, tile.S6},
		{"6s covers 9s", tile.S6, tile.S9, true, tile.S6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pond := []tile.Tile{{ID: tc.pondID}}
			if got := SujiSafe(pond, tile.Tile{ID: tc.safeID}); got != tc.wantSafe {
				t.Errorf("SujiSafe(pond=[%d], candidate=%d) = %v, want %v",
					tc.pondID, tc.safeID, got, tc.wantSafe)
			}
			// Ranks 4/5/6 are never suji-safe (no rank pair covers them).
			if SujiSafe(pond, tile.Tile{ID: tc.notSafeID}) {
				t.Errorf(
					"SujiSafe(pond=[%d], candidate=%d) = true, want false (rank 4/5/6 not suji-safe)",
					tc.pondID,
					tc.notSafeID,
				)
			}
		})
	}

	// Honor candidate: never suji-safe.
	pondP4 := []tile.Tile{{ID: tile.P4}}
	if SujiSafe(pondP4, tile.Tile{ID: tile.EastWind}) {
		t.Errorf("SujiSafe(pond=[4p], candidate=East) = true, want false (honors have no suji)")
	}
	if SujiSafe(pondP4, tile.Tile{ID: tile.Haku}) {
		t.Errorf("SujiSafe(pond=[4p], candidate=Haku) = true, want false (dragons have no suji)")
	}

	// Cross-suit: 4m in pond does NOT make 1p suji-safe.
	pondM4 := []tile.Tile{{ID: tile.M4}}
	if SujiSafe(pondM4, tile.Tile{ID: tile.P1}) {
		t.Errorf("SujiSafe(pond=[4m], candidate=1p) = true, want false (suji is per-suit)")
	}

	// Empty pond: nothing is suji-safe.
	if SujiSafe(nil, tile.Tile{ID: tile.M1}) {
		t.Errorf("SujiSafe(nil, 1m) = true, want false (empty pond)")
	}

	// Negative cases: pond doesn't contain required rank.
	pondM3 := []tile.Tile{{ID: tile.M3}}
	if SujiSafe(pondM3, tile.Tile{ID: tile.M1}) {
		t.Errorf("SujiSafe(pond=[3m], candidate=1m) = true, want false (3m doesn't cover 1m)")
	}
}

func TestBotDangerAwarePickDiscardPrefersGenbutsu(t *testing.T) {
	// Hand: M3..M7 (heavily connected, low isolation ~94 each) + EastWind
	// (honor, max isolation 1000). Without danger awareness, EastWind wins
	// hands-down on isolation. With M5 marked genbutsu (danger=0) and
	// everything else default (danger=2), the K=2000 penalty must dominate
	// the ~900 isolation gap and force the bot to discard M5.
	hand := []tile.Tile{
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.EastWind},
	}
	danger := map[uint8]int{tile.M5: 0}

	b := newDeterministicBot(SeatSouth, 1)
	idx := b.DangerAwarePickDiscard(hand, danger)
	if idx < 0 || idx >= len(hand) {
		t.Fatalf("DangerAwarePickDiscard returned out-of-range index %d", idx)
	}
	if hand[idx].ID != tile.M5 {
		t.Errorf(
			"DangerAwarePickDiscard chose %s, want 5m (genbutsu — danger 0 must beat unknown despite lower isolation)",
			hand[idx],
		)
	}
}

func TestBotDangerAwarePickDiscardPrefersSujiOverUnknown(t *testing.T) {
	// Hand: M1 (suji, danger 1) and P1 (unknown, danger 2). Both fully
	// isolated within their own suit (no neighbors anywhere in this hand,
	// no copies), so isolation scores tie at 100. The danger map alone
	// decides.
	hand := []tile.Tile{
		{ID: tile.M1},
		{ID: tile.P1},
	}
	danger := map[uint8]int{
		tile.M1: 1, // suji-safe against some opponent
		tile.P1: 2, // unknown danger
	}

	b := newDeterministicBot(SeatSouth, 1)
	idx := b.DangerAwarePickDiscard(hand, danger)
	if hand[idx].ID != tile.M1 {
		t.Errorf(
			"DangerAwarePickDiscard chose %s, want 1m (suji danger 1 beats unknown danger 2 at equal isolation)",
			hand[idx],
		)
	}
}

func TestBotDangerAwarePickDiscardFallsBackToIsolation(t *testing.T) {
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
	want := b.PickDiscard(hand)
	if got := b.DangerAwarePickDiscard(hand, nil); got != want {
		t.Errorf(
			"DangerAwarePickDiscard(hand, nil) = %d, want %d (PickDiscard fallback)",
			got,
			want,
		)
	}
	if got := b.DangerAwarePickDiscard(hand, map[uint8]int{}); got != want {
		t.Errorf(
			"DangerAwarePickDiscard(hand, empty) = %d, want %d (PickDiscard fallback)",
			got,
			want,
		)
	}
}

func TestFoldDiscardPicksGenbutsuOverHigherIsolation(t *testing.T) {
	// Fold-mode adversarial fixture: P5 is the genbutsu (danger 0); NorthWind
	// is an unknown-danger honor with max isolation. Push-mode (K=2000) might
	// pick P5 already, but the test confirms fold-mode does too with full
	// confidence: K=1_000_000 means danger ALWAYS wins regardless of isolation.
	hand := []tile.Tile{
		{ID: tile.P5},
		{ID: tile.NorthWind},
	}
	danger := map[uint8]int{tile.P5: 0}

	b := newDeterministicBot(SeatSouth, 1)
	idx := b.FoldDiscard(hand, danger)
	if idx < 0 || idx >= len(hand) {
		t.Fatalf("FoldDiscard returned out-of-range index %d", idx)
	}
	if hand[idx].ID != tile.P5 {
		t.Errorf(
			"FoldDiscard chose %s, want 5p (genbutsu must always win in fold mode)",
			hand[idx],
		)
	}
}

func TestFoldDiscardFallsBackToPickDiscardWhenDangerEmpty(t *testing.T) {
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
	want := b.PickDiscard(hand)
	if got := b.FoldDiscard(hand, nil); got != want {
		t.Errorf("FoldDiscard(hand, nil) = %d, want %d (PickDiscard fallback)", got, want)
	}
}
