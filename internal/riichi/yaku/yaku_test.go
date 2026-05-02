package yaku

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func mustHand(t *testing.T, s, winning string, opts ...handOption) hand.Hand {
	t.Helper()
	tiles, err := tile.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	w, err := tile.ParseOne(winning)
	if err != nil {
		t.Fatalf("parse winning %q: %v", winning, err)
	}
	h := hand.Hand{Concealed: tiles, Winning: w}
	for _, opt := range opts {
		opt(&h)
	}
	return h
}

type handOption func(h *hand.Hand)

func withTsumo() handOption { return func(h *hand.Hand) { h.IsTsumo = true } }
func withOpen() handOption  { return func(h *hand.Hand) { h.Open = true } }

func eastEastCtx() Context  { return Context{SeatWind: tile.EastWind, RoundWind: tile.EastWind} }
func southEastCtx() Context { return Context{SeatWind: tile.SouthWind, RoundWind: tile.EastWind} }

func haveYaku(matches []Match, name string) bool {
	for _, m := range matches {
		if m.Name == name {
			return true
		}
	}
	return false
}

func haveYakuPrefix(matches []Match, prefix string) bool {
	for _, m := range matches {
		if len(m.Name) >= len(prefix) && m.Name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func evaluateAll(h hand.Hand, ctx Context) [][]Match {
	decomps := hand.Decompose(h)
	out := make([][]Match, len(decomps))
	for i, d := range decomps {
		out[i] = Evaluate(d, h, ctx)
	}
	return out
}

func TestYaku_Riichi(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m")
	ctx := southEastCtx()
	ctx.Riichi = true
	for _, ms := range evaluateAll(h, ctx) {
		if haveYaku(ms, "Riichi") {
			return
		}
	}
	t.Error("expected Riichi")
}

func TestYaku_MenzenTsumo(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Menzen tsumo") {
			return
		}
	}
	t.Error("expected Menzen tsumo")
}

func TestYaku_MenzenTsumo_NotWhenOpen(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo(), withOpen())
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Menzen tsumo") {
			t.Error("Menzen tsumo must not match when open")
		}
	}
}

func TestYaku_Pinfu(t *testing.T) {
	// All sequences, pair 5p (non-yakuhai), winning 4m closes ryanmen on 5m6m
	h := mustHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m")
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Pinfu") {
			return
		}
	}
	t.Error("expected Pinfu")
}

func TestYaku_Pinfu_NotKanchan(t *testing.T) {
	// 4m_6m kanchan on 5m → no pinfu even though all sequences and non-yakuhai pair
	h := mustHand(t, "4m5m6m7p8p9p2s3s4s2p3p4p5p5p", "5m")
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Pinfu") {
			t.Error("Pinfu must not match for a kanchan-only completion")
		}
	}
}

func TestYaku_Pinfu_NotWhenYakuhaiPair(t *testing.T) {
	// All sequences but pair is round wind (1z) → no pinfu
	h := mustHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s1z1z", "4m")
	ctx := eastEastCtx()
	for _, ms := range evaluateAll(h, ctx) {
		if haveYaku(ms, "Pinfu") {
			t.Error("Pinfu must not match when pair is yakuhai")
		}
	}
}

func TestYaku_Pinfu_NotWhenOpen(t *testing.T) {
	h := mustHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m", withOpen())
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Pinfu") {
			t.Error("Pinfu must not match when open")
		}
	}
}

func TestYaku_Tanyao(t *testing.T) {
	h := mustHand(t, "2m3m4m5m6m7m2p3p4p5p6p7p8s8s", "4m")
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Tanyao") {
			return
		}
	}
	t.Error("expected Tanyao")
}

func TestYaku_Yakuhai_DoubleEastForDealerEast(t *testing.T) {
	// Dealer (seat E) in round E with East triplet → double yakuhai (2 han)
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1z1z1z2z2z", "9s")
	matches := flattenMatches(evaluateAll(h, eastEastCtx()))
	count := 0
	for _, m := range matches {
		if m.Name == "Yakuhai (round wind)" || m.Name == "Yakuhai (seat wind)" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected 2 yakuhai matches for double east, got %d (%v)", count, matches)
	}
}

func TestYaku_Yakuhai_NotForNonApplicableWind(t *testing.T) {
	// Seat S, round E, hand has West triplet → no yakuhai
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s3z3z3z2z2z", "9s")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if haveYakuPrefix(matches, "Yakuhai") {
		t.Errorf("expected no yakuhai for west triplet at seat S round E, got %v", matches)
	}
}

func TestYaku_Yakuhai_DragonAlwaysCounts(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s7z7z7z2z2z", "9s")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if !haveYakuPrefix(matches, "Yakuhai (Chun") {
		t.Errorf("expected dragon yakuhai (Chun), got %v", matches)
	}
}

func TestYaku_Iipeikou_Concealed(t *testing.T) {
	// Two identical 1m2m3m sequences
	h := mustHand(t, "1m2m3m1m2m3m4p5p6p7s8s9s2z2z", "4p")
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Iipeikou") {
			return
		}
	}
	t.Error("expected Iipeikou")
}

func TestYaku_Iipeikou_NotWhenOpen(t *testing.T) {
	h := mustHand(t, "1m2m3m1m2m3m4p5p6p7s8s9s2z2z", "4p", withOpen())
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Iipeikou") {
			t.Error("Iipeikou must not match when open")
		}
	}
}

func TestYaku_Iipeikou_NotInChiitoitsu(t *testing.T) {
	// Hand that admits both forms: standard with iipeikou shape, and chiitoitsu
	h := mustHand(t, "2m3m4m2m3m4m5p6p7p5p6p7p8s8s", "4m")
	// Direct check: find chiitoitsu decomp and verify no iipeikou.
	decomps := hand.Decompose(h)
	foundChiitoitsu := false
	for _, d := range decomps {
		if d.Form != hand.FormChiitoitsu {
			continue
		}
		foundChiitoitsu = true
		ms := Evaluate(d, h, southEastCtx())
		if haveYaku(ms, "Iipeikou") {
			t.Error("Iipeikou must not match within a chiitoitsu decomposition")
		}
	}
	if !foundChiitoitsu {
		t.Fatal("expected a chiitoitsu decomposition for this hand")
	}
}

func TestYaku_Toitoi(t *testing.T) {
	h := mustHand(t, "1m1m1m4m4m4m7p7p7p2s2s2s5p5p", "5p")
	for _, ms := range evaluateAll(h, southEastCtx()) {
		if haveYaku(ms, "Toitoi") {
			return
		}
	}
	t.Error("expected Toitoi")
}

func TestYaku_Honitsu_ConcealedThreeHan(t *testing.T) {
	h := mustHand(t, "1m2m3m4m5m6m7m8m9m1z1z1z2z2z", "9m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Honitsu" {
			if m.Han != 3 {
				t.Errorf("concealed Honitsu = %d han, want 3", m.Han)
			}
			return
		}
	}
	t.Error("expected Honitsu")
}

func TestYaku_Honitsu_OpenTwoHan(t *testing.T) {
	h := mustHand(t, "1m2m3m4m5m6m7m8m9m1z1z1z2z2z", "9m", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Honitsu" {
			if m.Han != 2 {
				t.Errorf("open Honitsu = %d han, want 2", m.Han)
			}
			return
		}
	}
	t.Error("expected Honitsu")
}

func TestYaku_SanshokuDoujun(t *testing.T) {
	h := mustHand(t, "1m2m3m1p2p3p1s2s3s5m6m7m8s8s", "7m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Sanshoku doujun" {
			if m.Han != 2 {
				t.Errorf("Sanshoku doujun = %d han, want 2 concealed", m.Han)
			}
			return
		}
	}
	t.Error("expected Sanshoku doujun")
}

func TestYaku_SanshokuDoujun_OpenDowngrade(t *testing.T) {
	h := mustHand(t, "1m2m3m1p2p3p1s2s3s5m6m7m8s8s", "7m", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Sanshoku doujun" {
			if m.Han != 1 {
				t.Errorf("Sanshoku doujun open = %d han, want 1", m.Han)
			}
			return
		}
	}
	t.Error("expected Sanshoku doujun")
}

func TestYaku_Ittsuu(t *testing.T) {
	h := mustHand(t, "1m2m3m4m5m6m7m8m9m1p2p3p5p5p", "9m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Ittsuu" {
			if m.Han != 2 {
				t.Errorf("Ittsuu = %d han, want 2 concealed", m.Han)
			}
			return
		}
	}
	t.Error("expected Ittsuu")
}

func TestYaku_Ittsuu_OpenDowngrade(t *testing.T) {
	h := mustHand(t, "1m2m3m4m5m6m7m8m9m1p2p3p5p5p", "9m", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Ittsuu" {
			if m.Han != 1 {
				t.Errorf("Ittsuu open = %d han, want 1", m.Han)
			}
			return
		}
	}
	t.Error("expected Ittsuu")
}

func TestYaku_PinfuTsumo_BothMatch(t *testing.T) {
	// Pinfu hand won by tsumo: both yaku present (fu-side flat-20 handled in score package)
	h := mustHand(t, "1m2m3m4m5m6m7p8p9p2s3s4s5p5p", "4m", withTsumo())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if !haveYaku(matches, "Pinfu") {
		t.Error("expected Pinfu")
	}
	if !haveYaku(matches, "Menzen tsumo") {
		t.Error("expected Menzen tsumo")
	}
}

func TestYaku_KokushiYakumanCapsOthers(t *testing.T) {
	h := mustHand(t, "1m9m1p9p1s9s1z2z3z4z5z6z7z1m", "1m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if len(matches) == 0 {
		t.Fatal("expected kokushi yakuman match")
	}
	for _, m := range matches {
		if !m.IsYakuman {
			t.Errorf("non-yakuman yaku slipped past yakuman cap: %v", m)
		}
	}
}

func flattenMatches(in [][]Match) []Match {
	var out []Match
	for _, ms := range in {
		out = append(out, ms...)
	}
	return out
}

// --- Group A fixtures.

func TestYaku_Chinitsu(t *testing.T) {
	h := mustHand(t, "1m1m1m4m4m4m7m7m7m9m9m9m5m5m", "5m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Chinitsu" {
			if m.Han != 6 {
				t.Errorf("Chinitsu = %d, want 6 concealed", m.Han)
			}
			return
		}
	}
	t.Error("expected Chinitsu")
}

func TestYaku_Chinitsu_OpenDowngrade(t *testing.T) {
	h := mustHand(t, "1m1m1m4m4m4m7m7m7m9m9m9m5m5m", "5m", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Chinitsu" {
			if m.Han != 5 {
				t.Errorf("Chinitsu open = %d, want 5", m.Han)
			}
			return
		}
	}
	t.Error("expected Chinitsu (open)")
}

func TestYaku_Chinitsu_RejectsHonors(t *testing.T) {
	// Same single-suit + honors hand used in TestYaku_Honitsu_ConcealedThreeHan
	h := mustHand(t, "1m2m3m4m5m6m7m8m9m1z1z1z2z2z", "9m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if haveYaku(matches, "Chinitsu") {
		t.Error("Chinitsu must not match a hand with honors")
	}
	if !haveYaku(matches, "Honitsu") {
		t.Error("Honitsu should still match the same hand")
	}
}

func TestYaku_Honroutou(t *testing.T) {
	h := mustHand(t, "1m1m1m9m9m9m1p1p1p9p9p1z1z1z", "1z")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if !haveYaku(matches, "Honroutou") {
		t.Error("expected Honroutou")
	}
	if !haveYaku(matches, "Toitoi") {
		t.Error(
			"Honroutou hand should also have Toitoi (no sequences possible with all yaochuhai tiles)",
		)
	}
}

func TestYaku_Chanta(t *testing.T) {
	h := mustHand(t, "1m2m3m7m8m9m1z1z1z9p9p9p2z2z", "1z")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Chanta" {
			if m.Han != 2 {
				t.Errorf("Chanta = %d, want 2 concealed", m.Han)
			}
			return
		}
	}
	t.Error("expected Chanta")
}

func TestYaku_Chanta_OpenDowngrade(t *testing.T) {
	h := mustHand(t, "1m2m3m7m8m9m1z1z1z9p9p9p2z2z", "1z", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Chanta" {
			if m.Han != 1 {
				t.Errorf("Chanta open = %d, want 1", m.Han)
			}
			return
		}
	}
	t.Error("expected Chanta (open)")
}

func TestYaku_Junchan(t *testing.T) {
	h := mustHand(t, "1m2m3m7m8m9m1p2p3p7p8p9p1s1s", "1s")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Junchan" {
			if m.Han != 3 {
				t.Errorf("Junchan = %d, want 3 concealed", m.Han)
			}
			return
		}
	}
	t.Error("expected Junchan")
}

// Junchan supersedes Chanta when no honors are present.
func TestYaku_Junchan_SupersedesChanta(t *testing.T) {
	h := mustHand(t, "1m2m3m7m8m9m1p2p3p7p8p9p1s1s", "1s")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if haveYaku(matches, "Chanta") {
		t.Error("Chanta must not match when no honors are present (Junchan supersedes)")
	}
	if !haveYaku(matches, "Junchan") {
		t.Error("expected Junchan")
	}
}

func TestYaku_Sanankou(t *testing.T) {
	// 3 concealed triplets + sequence + pair, winning tile in the sequence
	// (not on a shanpon completion). Sanankou matches whether ron or tsumo.
	h := mustHand(t, "1m1m1m4p4p4p7s7s7s2m3m4m5p5p", "4m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Sanankou" {
			if m.Han != 2 {
				t.Errorf("Sanankou = %d, want 2", m.Han)
			}
			return
		}
	}
	t.Error("expected Sanankou")
}

// Spec example: shanpon-ron downgrades the winning triplet to "open" for
// sanankou-counting. Hand has 3 triplets total (1m, 4m, 9m); winning 9m
// completes the 9m triplet via shanpon. On ron only 2 concealed triplets
// remain, suppressing sanankou. On tsumo all 3 stay concealed.
func TestYaku_Sanankou_ShanponRonSuppresses(t *testing.T) {
	hRon := mustHand(t, "1m1m1m4m4m4m1p2p3p9m9m9m5p5p", "9m")
	if haveYaku(flattenMatches(evaluateAll(hRon, southEastCtx())), "Sanankou") {
		t.Error("Sanankou must not match on ron when winning tile completes a triplet via shanpon")
	}

	hTsumo := mustHand(t, "1m1m1m4m4m4m1p2p3p9m9m9m5p5p", "9m", withTsumo())
	if !haveYaku(flattenMatches(evaluateAll(hTsumo, southEastCtx())), "Sanankou") {
		t.Error("Sanankou must match on tsumo (all 3 triplets concealed)")
	}
}

func TestYaku_SanshokuDoukou(t *testing.T) {
	h := mustHand(t, "4m4m4m4p4p4p4s4s4s1m2m3m5p5p", "3m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Sanshoku doukou" {
			if m.Han != 2 {
				t.Errorf("Sanshoku doukou = %d, want 2", m.Han)
			}
			return
		}
	}
	t.Error("expected Sanshoku doukou")
}

func TestYaku_Shousangen(t *testing.T) {
	h := mustHand(t, "5z5z5z6z6z6z7z7z1m2m3m4p5p6p", "3m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Shousangen" {
			if m.Han != 2 {
				t.Errorf("Shousangen = %d, want 2", m.Han)
			}
			return
		}
	}
	t.Error("expected Shousangen")
}

func TestYaku_Shousangen_AnyDragonAsPair(t *testing.T) {
	cases := []struct {
		name string
		hand string
	}{
		{"haku pair (5z), hatsu+chun triplets", "6z6z6z7z7z7z5z5z1m2m3m4p5p6p"},
		{"hatsu pair (6z), haku+chun triplets", "5z5z5z7z7z7z6z6z1m2m3m4p5p6p"},
		{"chun pair (7z), haku+hatsu triplets", "5z5z5z6z6z6z7z7z1m2m3m4p5p6p"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := mustHand(t, c.hand, "3m")
			matches := flattenMatches(evaluateAll(h, southEastCtx()))
			if !haveYaku(matches, "Shousangen") {
				t.Errorf("expected Shousangen for %s", c.name)
			}
		})
	}
}

func TestYaku_Ryanpeikou(t *testing.T) {
	h := mustHand(t, "1m2m3m1m2m3m5p6p7p5p6p7p8s8s", "3m")
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	for _, m := range matches {
		if m.Name == "Ryanpeikou" {
			if m.Han != 3 {
				t.Errorf("Ryanpeikou = %d, want 3", m.Han)
			}
			return
		}
	}
	t.Error("expected Ryanpeikou")
}

func TestYaku_Ryanpeikou_SuppressesIipeikou(t *testing.T) {
	h := mustHand(t, "1m2m3m1m2m3m5p6p7p5p6p7p8s8s", "3m")
	for _, d := range hand.Decompose(h) {
		if d.Form != hand.FormStandard {
			continue
		}
		ms := Evaluate(d, h, southEastCtx())
		if haveYaku(ms, "Ryanpeikou") && haveYaku(ms, "Iipeikou") {
			t.Error("Iipeikou must be suppressed when Ryanpeikou matches the same decomposition")
		}
	}
}

func TestYaku_Ryanpeikou_NotWhenOpen(t *testing.T) {
	h := mustHand(t, "1m2m3m1m2m3m5p6p7p5p6p7p8s8s", "3m", withOpen())
	matches := flattenMatches(evaluateAll(h, southEastCtx()))
	if haveYaku(matches, "Ryanpeikou") {
		t.Error("Ryanpeikou must not match an open hand")
	}
}

// --- Group C: situational / turn-aware yaku.

func TestYaku_Ippatsu_MatchesWhenFlagSetAndConcealed(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx()
	ctx.Riichi = true
	ctx.Ippatsu = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYaku(matches, "Ippatsu") {
		t.Errorf("expected Ippatsu in matches: %+v", matches)
	}
}

func TestYaku_Ippatsu_NoMatchWhenFlagOff(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx()
	ctx.Riichi = true
	ctx.Ippatsu = false
	matches := flattenMatches(evaluateAll(h, ctx))
	if haveYaku(matches, "Ippatsu") {
		t.Errorf("Ippatsu must not match when flag is off: %+v", matches)
	}
}

func TestYaku_Ippatsu_NoMatchWhenOpen(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo(), withOpen())
	ctx := southEastCtx()
	ctx.Riichi = true
	ctx.Ippatsu = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if haveYaku(matches, "Ippatsu") {
		t.Errorf("Ippatsu must not match an open hand even with flag on: %+v", matches)
	}
}

func TestYaku_DoubleRiichi_SuppressesRegularRiichi(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m")
	ctx := southEastCtx()
	ctx.Riichi = true
	ctx.DoubleRiichi = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYaku(matches, "Double riichi") {
		t.Errorf("expected Double riichi in matches: %+v", matches)
	}
	if haveYaku(matches, "Riichi") {
		t.Errorf("Riichi must not match when Double riichi matches: %+v", matches)
	}
}

func TestYaku_Haitei_MatchesOnlyOnTsumo(t *testing.T) {
	tsumoHand := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx()
	ctx.Haitei = true
	matches := flattenMatches(evaluateAll(tsumoHand, ctx))
	if !haveYakuPrefix(matches, "Haitei") {
		t.Errorf("expected Haitei on tsumo: %+v", matches)
	}

	ronHand := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m") // ron (no IsTsumo)
	matches = flattenMatches(evaluateAll(ronHand, ctx))
	if haveYakuPrefix(matches, "Haitei") {
		t.Errorf("Haitei must not match on ron: %+v", matches)
	}
}

func TestYaku_Houtei_MatchesOnlyOnRon(t *testing.T) {
	ronHand := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m")
	ctx := southEastCtx()
	ctx.Houtei = true
	matches := flattenMatches(evaluateAll(ronHand, ctx))
	if !haveYakuPrefix(matches, "Houtei") {
		t.Errorf("expected Houtei on ron: %+v", matches)
	}

	tsumoHand := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	matches = flattenMatches(evaluateAll(tsumoHand, ctx))
	if haveYakuPrefix(matches, "Houtei") {
		t.Errorf("Houtei must not match on tsumo: %+v", matches)
	}
}

func TestYaku_Tenhou_DealerOnlyYakuman(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := eastEastCtx() // dealer
	ctx.Tenhou = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYaku(matches, "Tenhou") {
		t.Errorf("expected Tenhou for dealer with flag set: %+v", matches)
	}
}

func TestYaku_Tenhou_NoMatchForNonDealer(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx() // non-dealer
	ctx.Tenhou = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if haveYaku(matches, "Tenhou") {
		t.Errorf("Tenhou must not match for non-dealer seat: %+v", matches)
	}
}

func TestYaku_Chiihou_NonDealerYakuman(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx() // non-dealer
	ctx.Chiihou = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYaku(matches, "Chiihou") {
		t.Errorf("expected Chiihou for non-dealer with flag set: %+v", matches)
	}
}

func TestYaku_Chiihou_NoMatchForDealer(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := eastEastCtx() // dealer
	ctx.Chiihou = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if haveYaku(matches, "Chiihou") {
		t.Errorf("Chiihou must not match for dealer seat: %+v", matches)
	}
}

func TestYaku_Rinshan_DormantUntilFlagSet(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m", withTsumo())
	ctx := southEastCtx()
	ctx.Rinshan = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYakuPrefix(matches, "Rinshan") {
		t.Errorf("Rinshan detector must match when flag forced on: %+v", matches)
	}

	ctx.Rinshan = false
	matches = flattenMatches(evaluateAll(h, ctx))
	if haveYakuPrefix(matches, "Rinshan") {
		t.Errorf("Rinshan must not match when flag is off (always false in v1): %+v", matches)
	}
}

func TestYaku_Chankan_DormantUntilFlagSet(t *testing.T) {
	h := mustHand(t, "1m2m3m4p5p6p7s8s9s1m1m2z2z2z", "1m")
	ctx := southEastCtx()
	ctx.Chankan = true
	matches := flattenMatches(evaluateAll(h, ctx))
	if !haveYaku(matches, "Chankan") {
		t.Errorf("Chankan detector must match when flag forced on: %+v", matches)
	}

	ctx.Chankan = false
	matches = flattenMatches(evaluateAll(h, ctx))
	if haveYaku(matches, "Chankan") {
		t.Errorf("Chankan must not match when flag is off (always false in v1): %+v", matches)
	}
}
