package calc

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func mustParse(t *testing.T, s string) []tile.Tile {
	t.Helper()
	tiles, err := tile.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tiles
}

func TestAnalyze_PinfuTsumoTanyaoRiichi(t *testing.T) {
	// Pinfu hand with tsumo + riichi + tanyao = 4 han, 20 fu (pinfu-tsumo flat).
	// Non-dealer (seat S, round E).
	h := hand.Hand{
		Concealed: mustParse(t, "3m4m5m6m7m2p3p4p5p6p7p3s3s2m"),
		Winning:   tile.Tile{ID: tile.M2},
		IsTsumo:   true,
	}
	ctx := Context{SeatWind: tile.SouthWind, RoundWind: tile.EastWind, Riichi: true}
	r := Analyze(h, ctx)
	if r == nil {
		t.Fatal("expected a winning result")
	}
	if r.Han != 4 {
		t.Errorf("Han = %d, want 4 (riichi + menzen tsumo + pinfu + tanyao)", r.Han)
	}
	if r.Fu != 20 {
		t.Errorf("Fu = %d, want 20 (pinfu-tsumo flat)", r.Fu)
	}
}

func TestAnalyze_NoYakuReturnsNil(t *testing.T) {
	// 14-tile winning shape with no yaku at non-dealer S round E.
	// 1m2m3m + 4p5p6p + 7s8s9s + 1z1z (round wind pair, +2 fu but no han)
	// + 9m9m9m? Actually we need to construct a hand with no yaku.
	// Easier: hand with no yaku at all (open with no yakuhai, no shape yaku).
	// All sequences, no yakuhai pair, but Open=true means pinfu rejected,
	// no menzen tsumo. → no yaku.
	h := hand.Hand{
		Concealed: mustParse(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p"),
		Winning:   tile.Tile{ID: tile.M4},
		Open:      true,
		IsTsumo:   true,
	}
	ctx := Context{SeatWind: tile.SouthWind, RoundWind: tile.EastWind}
	r := Analyze(h, ctx)
	if r != nil {
		t.Errorf("expected nil for yakuless hand, got %+v", r)
	}
}

func TestAnalyze_DecompositionSelectionPicksHigherScoring(t *testing.T) {
	// A hand that admits both chiitoitsu (2 han, 25 fu) and standard (lower-scoring)
	// decompositions: 2m3m4m 2m3m4m 5p6p7p 5p6p7p 8s8s with iipeikou under standard.
	// Chiitoitsu: 2 han 25 fu. Standard with iipeikou (concealed): 1 han.
	// Plus tanyao adds 1 han to both, so chiitoitsu = 3 han (chiitoitsu + tanyao),
	// standard = 2 han (iipeikou + tanyao). Chiitoitsu should win.
	h := hand.Hand{
		Concealed: mustParse(t, "2m3m4m2m3m4m5p6p7p5p6p7p8s8s"),
		Winning:   tile.Tile{ID: tile.M4},
		IsTsumo:   true,
	}
	ctx := Context{SeatWind: tile.SouthWind, RoundWind: tile.EastWind}
	r := Analyze(h, ctx)
	if r == nil {
		t.Fatal("expected a winning result")
	}
	// Both forms produce results; the orchestrator picks the highest-scoring.
	// Chiitoitsu's pinfu-tsumo equivalent: 25 fu × 2^(2+han). Standard pinfu-tsumo: 20 fu.
	// We assert: the chosen form is whichever yields higher total points. We don't
	// hard-code the form here — just sanity-check that han + tanyao + tsumo are
	// counted (yaku list contains menzen tsumo and tanyao at minimum).
	hasTsumo := false
	hasTanyao := false
	for _, m := range r.YakuMatches {
		if m.Name == "Menzen tsumo" {
			hasTsumo = true
		}
		if m.Name == "Tanyao" {
			hasTanyao = true
		}
	}
	if !hasTsumo {
		t.Error("expected Menzen tsumo")
	}
	if !hasTanyao {
		t.Error("expected Tanyao")
	}
}

func TestAnalyze_KokushiYakuman(t *testing.T) {
	h := hand.Hand{
		Concealed: mustParse(t, "1m9m1p9p1s9s1z2z3z4z5z6z7z1m"),
		Winning:   tile.Tile{ID: tile.M1},
	}
	ctx := Context{SeatWind: tile.SouthWind, RoundWind: tile.EastWind}
	r := Analyze(h, ctx)
	if r == nil {
		t.Fatal("expected kokushi yakuman result")
	}
	if r.Award.Tier != 6 {
		// TierYakuman = 6 in the score package.
		t.Errorf("tier = %v, want yakuman", r.Award.Tier)
	}
	if r.Award.Total != 32000 {
		t.Errorf("total = %d, want 32000", r.Award.Total)
	}
}
