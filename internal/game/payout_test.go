package game

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
)

func mockRonResult(total, base int) *calc.Result {
	return &calc.Result{Award: score.Award{Total: total, Base: base}}
}

func TestComputePayoutsNonDealerRon30Fu3HanWithHonba2(t *testing.T) {
	// 30fu 3han non-dealer ron: base = 30 * 2^(2+3) = 960; total = roundUp100(960*4) = 3900.
	result := mockRonResult(3900, 960)
	o := OutcomeRon{Winner: SeatSouth, Loser: SeatNorth, Result: result}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 2, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{0, +4500, 0, -4500} // E, S, W, N — South gains, North pays (3900 + 300*2 honba)
	if got != want {
		t.Errorf("ComputePayouts non-dealer ron 30fu/3han honba=2 = %v, want %v", got, want)
	}
}

func TestComputePayoutsDealerTsumoMangan(t *testing.T) {
	// Dealer tsumo mangan: base = 2000. Each non-dealer pays roundUp100(2000*2) = 4000.
	// Total to dealer = 12000 + 1000 stick = 13000.
	result := mockRonResult(12000, 2000)
	o := OutcomeTsumo{Winner: SeatEast, Result: result}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 1}

	got := ComputePayouts(o, ctx)
	want := [4]int{+13000, -4000, -4000, -4000}
	if got != want {
		t.Errorf("ComputePayouts dealer-tsumo mangan = %v, want %v", got, want)
	}
}

func TestComputePayoutsNonDealerTsumoMangan(t *testing.T) {
	// Non-dealer tsumo mangan: base = 2000.
	// Each non-dealer (other than winner) pays roundUp100(2000*1) = 2000.
	// Dealer pays roundUp100(2000*2) = 4000.
	// Winner gains 4000 + 2000 + 2000 = 8000.
	result := mockRonResult(8000, 2000)
	o := OutcomeTsumo{Winner: SeatSouth, Result: result}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{-4000, +8000, -2000, -2000}
	if got != want {
		t.Errorf("ComputePayouts non-dealer-tsumo mangan = %v, want %v", got, want)
	}
}

func TestComputePayoutsDealerTsumoWithHonba(t *testing.T) {
	// Dealer tsumo mangan with honba=3: each non-dealer pays 4000 + 100*3 = 4300.
	// Winner gains 4300 * 3 = 12900 + 0 sticks.
	result := mockRonResult(12000, 2000)
	o := OutcomeTsumo{Winner: SeatEast, Result: result}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 3, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{+12900, -4300, -4300, -4300}
	if got != want {
		t.Errorf("ComputePayouts dealer-tsumo mangan honba=3 = %v, want %v", got, want)
	}
}

func TestComputePayoutsRyuukyokuOneTenpai(t *testing.T) {
	// 1 tenpai (North), 3 noten. Each noten pays 1000, tenpai gains 3000.
	o := OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatNorth}}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{-1000, -1000, -1000, +3000}
	if got != want {
		t.Errorf("ComputePayouts ryuukyoku 1-tenpai = %v, want %v", got, want)
	}
}

func TestComputePayoutsRyuukyokuTwoTenpai(t *testing.T) {
	// 2 tenpai (East+West), 2 noten (South+North). Each noten pays 1500, each tenpai gains 1500.
	o := OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatWest}}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{+1500, -1500, +1500, -1500}
	if got != want {
		t.Errorf("ComputePayouts ryuukyoku 2-tenpai = %v, want %v", got, want)
	}
}

func TestComputePayoutsRyuukyokuThreeTenpai(t *testing.T) {
	// 3 tenpai (East+South+West), 1 noten (North). Noten pays 3000; each tenpai gains 1000.
	o := OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatSouth, SeatWest}}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{+1000, +1000, +1000, -3000}
	if got != want {
		t.Errorf("ComputePayouts ryuukyoku 3-tenpai = %v, want %v", got, want)
	}
}

func TestComputePayoutsRyuukyokuAllTenpai(t *testing.T) {
	o := OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatSouth, SeatWest, SeatNorth}}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{0, 0, 0, 0}
	if got != want {
		t.Errorf("ComputePayouts ryuukyoku all-tenpai = %v, want %v (no transfer)", got, want)
	}
}

func TestComputePayoutsRyuukyokuAllNoten(t *testing.T) {
	o := OutcomeRyuukyoku{TenpaiPlayers: nil}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}

	got := ComputePayouts(o, ctx)
	want := [4]int{0, 0, 0, 0}
	if got != want {
		t.Errorf("ComputePayouts ryuukyoku all-noten = %v, want %v (no transfer)", got, want)
	}
}

func TestComputePayoutsRiichiStickSweep(t *testing.T) {
	// Non-dealer ron with 2 sticks pooled. Winner gets total + honba + 2*1000.
	result := mockRonResult(3900, 960)
	o := OutcomeRon{Winner: SeatSouth, Loser: SeatNorth, Result: result}
	ctx := PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 2}

	got := ComputePayouts(o, ctx)
	want := [4]int{0, +3900 + 2000, 0, -3900}
	if got != want {
		t.Errorf(
			"ComputePayouts ron with 2 sticks = %v, want %v (winner sweeps the pool)",
			got,
			want,
		)
	}
}
