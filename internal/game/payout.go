package game

// PayoutContext is the per-hand context the payout function needs:
// who the dealer is (for tsumo split), how many honba have accumulated
// (each adds 100 per payer), and how many riichi sticks are pooled
// (the agari winner sweeps them).
type PayoutContext struct {
	Dealer       Seat
	Honba        int
	RiichiSticks int
}

// ComputePayouts returns per-seat point deltas (positive = received,
// negative = paid) for the given outcome under the given context. The
// function is pure — it does not mutate its inputs.
//
// Outcome rules:
//   - OutcomeRon: winner gains Total + 300*Honba + 1000*RiichiSticks; loser
//     pays Total + 300*Honba.
//   - OutcomeTsumo: per Award.Base, derive per-payer payments. Dealer
//     winner: each non-dealer pays roundUp100(Base*2) + 100*Honba.
//     Non-dealer winner: each non-dealer (other than winner) pays
//     roundUp100(Base*1) + 100*Honba; dealer pays roundUp100(Base*2) +
//     100*Honba. Winner gains the sum + 1000*RiichiSticks.
//   - OutcomeRyuukyoku: 0 or 4 tenpai → no transfer. Otherwise total
//     transfer is 3000: each noten pays 3000/notenCount, each tenpai
//     receives 3000/tenpaiCount. Honba does NOT add to ryuukyoku.
func ComputePayouts(o Outcome, ctx PayoutContext) [4]int {
	var deltas [4]int
	switch v := o.(type) {
	case OutcomeRon:
		base := v.Result.Award.Total
		honbaBonus := 300 * ctx.Honba
		deltas[v.Winner] = base + honbaBonus + 1000*ctx.RiichiSticks
		deltas[v.Loser] = -(base + honbaBonus)
	case OutcomeTsumo:
		base := v.Result.Award.Base
		honbaPerPayer := 100 * ctx.Honba
		dealerPay := payoutRoundUp100(base*2) + honbaPerPayer
		nonDealerPay := payoutRoundUp100(base*1) + honbaPerPayer
		isDealerWinner := v.Winner == ctx.Dealer
		var winnerGain int
		for s := range Seat(numSeats) {
			if s == v.Winner {
				continue
			}
			var pay int
			switch {
			case isDealerWinner:
				pay = dealerPay
			case s == ctx.Dealer:
				pay = dealerPay
			default:
				pay = nonDealerPay
			}
			deltas[s] = -pay
			winnerGain += pay
		}
		deltas[v.Winner] = winnerGain + 1000*ctx.RiichiSticks
	case OutcomeRyuukyoku:
		tenpaiCount := len(v.TenpaiPlayers)
		notenCount := numSeats - tenpaiCount
		if tenpaiCount == 0 || tenpaiCount == numSeats {
			return deltas
		}
		tenpaiSet := make(map[Seat]bool, tenpaiCount)
		for _, s := range v.TenpaiPlayers {
			tenpaiSet[s] = true
		}
		tenpaiGain := 3000 / tenpaiCount
		notenLoss := 3000 / notenCount
		for s := range Seat(numSeats) {
			if tenpaiSet[s] {
				deltas[s] = tenpaiGain
			} else {
				deltas[s] = -notenLoss
			}
		}
	}
	return deltas
}

// payoutRoundUp100 rounds n up to the nearest 100. Duplicates the helper in
// internal/riichi/score for package self-containment — payout logic doesn't
// depend on the score package's internal helpers.
func payoutRoundUp100(n int) int {
	if n%100 == 0 {
		return n
	}
	return ((n / 100) + 1) * 100
}
