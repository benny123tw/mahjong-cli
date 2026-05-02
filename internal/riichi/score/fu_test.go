package score

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func mustParseHand(t *testing.T, s, winning string) hand.Hand {
	t.Helper()
	tiles, err := tile.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	w, err := tile.ParseOne(winning)
	if err != nil {
		t.Fatalf("parse winning %q: %v", winning, err)
	}
	return hand.Hand{Concealed: tiles, Winning: w}
}

func standardDecomp(t *testing.T, h hand.Hand) hand.Decomposition {
	t.Helper()
	for _, d := range hand.Decompose(h) {
		if d.Form == hand.FormStandard {
			return d
		}
	}
	t.Fatal("no standard decomposition")
	return hand.Decomposition{}
}

func chiitoitsuDecomp(t *testing.T, h hand.Hand) hand.Decomposition {
	t.Helper()
	for _, d := range hand.Decompose(h) {
		if d.Form == hand.FormChiitoitsu {
			return d
		}
	}
	t.Fatal("no chiitoitsu decomposition")
	return hand.Decomposition{}
}

func ctxEastEast() Context {
	return Context{SeatWind: tile.EastWind, RoundWind: tile.EastWind}
}

func TestFu_PinfuTsumoFlat20(t *testing.T) {
	h := mustParseHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m")
	h.IsTsumo = true
	d := standardDecomp(t, h)
	ctx := ctxEastEast()
	ctx.IsPinfu = true
	if got := Fu(d, h, ctx); got != 20 {
		t.Errorf("Fu(pinfu-tsumo) = %d, want 20", got)
	}
}

func TestFu_ChiitoitsuFlat25(t *testing.T) {
	h := mustParseHand(t, "1m1m4p4p7p7p2s2s5s5s8s8s1z1z", "1z")
	d := chiitoitsuDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 25 {
		t.Errorf("Fu(chiitoitsu) = %d, want 25", got)
	}
}

func TestFu_PinfuRon30(t *testing.T) {
	// Same pinfu shape but ron: 20 base + 10 menzen ron = 30
	h := mustParseHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m")
	d := standardDecomp(t, h)
	ctx := ctxEastEast()
	ctx.IsPinfu = true
	if got := Fu(d, h, ctx); got != 30 {
		t.Errorf("Fu(pinfu-ron) = %d, want 30", got)
	}
}

func TestFu_ConcealedTerminalTripletsRound(t *testing.T) {
	// Hand with two concealed yaochuhai triplets (1z, 2z), ron, menzen
	// Expected: 20 base + 10 menzen ron + 8 (1z triplet) + 8 (2z triplet)
	// + 2 (yakuhai pair? no — pair is 9s) = 46 → 50 after round-up.
	// Wait, 1z and 2z are yakuhai (round wind) at east-east — that adds 2 fu per
	// pair; but here 1z and 2z are TRIPLETS, not pair. The pair is 9s9s = no
	// yakuhai. So fu = 20 + 10 + 8 + 8 = 46 → 50.
	h := mustParseHand(t, "1m2m3m4p5p6p1z1z1z2z2z2z9s9s", "9s")
	d := standardDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 50 {
		t.Errorf("Fu(2 yaochuhai triplets ron) = %d, want 50", got)
	}
}

func TestFu_KanchanWait(t *testing.T) {
	// 4m_6m kanchan completed by 5m. Other sets not contributing fu.
	// 4m5m6m + 7p8p9p + 2s3s4s + 2p3p4p + 5p5p, winning = 5m at pos 1 of 4m5m6m.
	// Concealed ron: 20 base + 10 menzen + 2 kanchan = 32 → 40.
	h := mustParseHand(t, "4m5m6m7p8p9p2s3s4s2p3p4p5p5p", "5m")
	d := standardDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 40 {
		t.Errorf("Fu(kanchan ron) = %d, want 40 (32 rounded)", got)
	}
}

func TestFu_TankiWait(t *testing.T) {
	// 4 sequences + tanki on 1z. Concealed ron.
	// 20 base + 10 menzen ron + 2 tanki = 32 → 40.
	h := mustParseHand(t, "1m2m3m4m5m6m7p8p9p1s2s3s1z1z", "1z")
	d := standardDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 40 {
		t.Errorf("Fu(tanki ron) = %d, want 40", got)
	}
}

func TestFu_KuipinfuOpenBaseTwentyTwo(t *testing.T) {
	// Open hand with all sequences and non-yakuhai pair, winning ryanmen ron.
	// 20 base + 0 (no menzen, no other fu) → kuipinfu adds +2 → 22 → 30.
	h := mustParseHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m")
	h.Open = true
	d := standardDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 30 {
		t.Errorf("Fu(kuipinfu ron) = %d, want 30 (22 → 30)", got)
	}
}

func TestFu_RoundUp32To40(t *testing.T) {
	// Engineered: 20 base + 10 menzen + 2 kanchan = 32. Should round up to 40.
	// Same as TestFu_KanchanWait above.
	h := mustParseHand(t, "4m5m6m7p8p9p2s3s4s2p3p4p5p5p", "5m")
	d := standardDecomp(t, h)
	if got := Fu(d, h, ctxEastEast()); got != 40 {
		t.Errorf("Fu rounds 32 → %d, want 40", got)
	}
}
