package score

import "fmt"

type Tier int

const (
	TierNormal Tier = iota
	TierMangan
	TierHaneman
	TierBaiman
	TierSanbaiman
	TierKazoeYakuman
	TierYakuman
)

func (t Tier) String() string {
	switch t {
	case TierNormal:
		return "normal"
	case TierMangan:
		return "mangan"
	case TierHaneman:
		return "haneman"
	case TierBaiman:
		return "baiman"
	case TierSanbaiman:
		return "sanbaiman"
	case TierKazoeYakuman:
		return "kazoe yakuman"
	case TierYakuman:
		return "yakuman"
	}
	return "?"
}

type Award struct {
	Han       int
	Fu        int
	Base      int
	Tier      Tier
	Total     int
	Breakdown string
}

// Compute returns the score award for a winning hand. When isYakuman is true,
// the han value is ignored and the payout is fixed at the yakuman base (8000).
func Compute(han, fu int, isDealer, isTsumo, isYakuman bool) Award {
	var base int
	var tier Tier

	switch {
	case isYakuman:
		base = 8000
		tier = TierYakuman
	case han >= 13:
		base = 8000
		tier = TierKazoeYakuman
	case han >= 11:
		base = 6000
		tier = TierSanbaiman
	case han >= 8:
		base = 4000
		tier = TierBaiman
	case han >= 6:
		base = 3000
		tier = TierHaneman
	case han >= 5:
		base = 2000
		tier = TierMangan
	default:
		base = fu * (1 << (2 + han))
		if base >= 2000 {
			base = 2000
			tier = TierMangan
		} else {
			tier = TierNormal
		}
	}

	a := Award{Han: han, Fu: fu, Base: base, Tier: tier}
	if isTsumo {
		if isDealer {
			each := roundUp100(base * 2)
			a.Total = each * 3
			a.Breakdown = fmt.Sprintf("dealer tsumo: %d all", each)
		} else {
			nonDealerPay := roundUp100(base * 1)
			dealerPay := roundUp100(base * 2)
			a.Total = nonDealerPay*2 + dealerPay
			a.Breakdown = fmt.Sprintf("non-dealer tsumo: %d/%d", nonDealerPay, dealerPay)
		}
	} else {
		if isDealer {
			a.Total = roundUp100(base * 6)
			a.Breakdown = fmt.Sprintf("dealer ron: %d", a.Total)
		} else {
			a.Total = roundUp100(base * 4)
			a.Breakdown = fmt.Sprintf("non-dealer ron: %d", a.Total)
		}
	}
	return a
}

func roundUp100(n int) int {
	if n%100 == 0 {
		return n
	}
	return ((n / 100) + 1) * 100
}
