package score

import "testing"

func TestScore_NonDealerRonBoundaries(t *testing.T) {
	cases := []struct {
		han, fu  int
		want     int
		wantTier Tier
	}{
		{1, 30, 1000, TierNormal},
		{3, 30, 3900, TierNormal},
		{4, 30, 7700, TierNormal},
		{5, 30, 8000, TierMangan},
		{6, 30, 12000, TierHaneman},
		{7, 30, 12000, TierHaneman},
		{8, 30, 16000, TierBaiman},
		{10, 30, 16000, TierBaiman},
		{11, 30, 24000, TierSanbaiman},
		{12, 30, 24000, TierSanbaiman},
		{13, 30, 32000, TierKazoeYakuman},
	}
	for _, c := range cases {
		a := Compute(c.han, c.fu, false, false, false)
		if a.Total != c.want {
			t.Errorf(
				"Compute(han=%d, fu=%d, non-dealer ron) = %d, want %d",
				c.han,
				c.fu,
				a.Total,
				c.want,
			)
		}
		if a.Tier != c.wantTier {
			t.Errorf("Compute(han=%d, fu=%d) tier = %v, want %v", c.han, c.fu, a.Tier, c.wantTier)
		}
	}
}

func TestScore_DealerRon(t *testing.T) {
	a := Compute(3, 30, true, false, false)
	if a.Total != 5800 {
		t.Errorf("dealer ron 3han 30fu = %d, want 5800", a.Total)
	}
}

func TestScore_NonDealerTsumoSplit(t *testing.T) {
	// 2 han 40 fu → base 640. NDP=700, DP=1300, total=2700.
	a := Compute(2, 40, false, true, false)
	if a.Total != 2700 {
		t.Errorf("non-dealer tsumo 2han 40fu total = %d, want 2700", a.Total)
	}
	if a.Breakdown != "non-dealer tsumo: 700/1300" {
		t.Errorf("breakdown = %q, want %q", a.Breakdown, "non-dealer tsumo: 700/1300")
	}
}

func TestScore_DealerTsumoMangan(t *testing.T) {
	// Dealer tsumo mangan: each non-dealer pays 4000, total 12000.
	a := Compute(5, 30, true, true, false)
	if a.Total != 12000 {
		t.Errorf("dealer tsumo mangan total = %d, want 12000", a.Total)
	}
}

func TestScore_TrueYakumanCapsHan(t *testing.T) {
	// Yakuman: han is ignored. Non-dealer ron yakuman = 32000.
	a := Compute(20, 50, false, false, true)
	if a.Total != 32000 {
		t.Errorf("non-dealer yakuman = %d, want 32000", a.Total)
	}
	if a.Tier != TierYakuman {
		t.Errorf("tier = %v, want yakuman", a.Tier)
	}
}

func TestScore_DealerYakuman(t *testing.T) {
	a := Compute(13, 30, true, false, true)
	if a.Total != 48000 {
		t.Errorf("dealer yakuman = %d, want 48000", a.Total)
	}
}

func TestScore_KazoeYakumanFromThirteenHan(t *testing.T) {
	a := Compute(13, 30, false, false, false)
	if a.Tier != TierKazoeYakuman {
		t.Errorf("13 han non-yakuman tier = %v, want kazoe yakuman", a.Tier)
	}
	if a.Total != 32000 {
		t.Errorf("kazoe yakuman = %d, want 32000", a.Total)
	}
}

func TestScore_RoundUpTo100(t *testing.T) {
	if got := roundUp100(960); got != 1000 {
		t.Errorf("roundUp100(960) = %d, want 1000", got)
	}
	if got := roundUp100(1000); got != 1000 {
		t.Errorf("roundUp100(1000) = %d, want 1000", got)
	}
	if got := roundUp100(3840); got != 3900 {
		t.Errorf("roundUp100(3840) = %d, want 3900", got)
	}
}
