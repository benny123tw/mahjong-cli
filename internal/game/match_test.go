package game

import (
	"errors"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func mockResult(total, base int) *calc.Result {
	return &calc.Result{Award: score.Award{Total: total, Base: base}}
}

func TestNewMatchInitialState(t *testing.T) {
	m := NewMatch(7)

	want := [numSeats]int{25000, 25000, 25000, 25000}
	if got := m.Scores(); got != want {
		t.Errorf("NewMatch.Scores() = %v, want %v", got, want)
	}
	if m.Dealer() != SeatEast {
		t.Errorf("NewMatch.Dealer() = %d, want SeatEast", m.Dealer())
	}
	if m.HandIndex() != 0 {
		t.Errorf("NewMatch.HandIndex() = %d, want 0", m.HandIndex())
	}
	if m.Honba() != 0 {
		t.Errorf("NewMatch.Honba() = %d, want 0", m.Honba())
	}
	if m.RoundWind() != tile.EastWind {
		t.Errorf("NewMatch.RoundWind() = %d, want EastWind", m.RoundWind())
	}
	if m.RiichiSticks() != 0 {
		t.Errorf("NewMatch.RiichiSticks() = %d, want 0", m.RiichiSticks())
	}
	if m.CurrentGame() == nil {
		t.Errorf("NewMatch.CurrentGame() is nil, want non-nil")
	}
	if m.IsFinished() {
		t.Errorf("NewMatch.IsFinished() = true, want false")
	}
}

func TestAdvanceFromOutcomeNonDealerRonRotates(t *testing.T) {
	m := NewMatch(7)
	// 30fu 1han non-dealer ron: base = 30 * 8 = 240; total = roundUp100(240*4) = 1000.
	o := OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: mockResult(1000, 240)}

	tr, err := m.AdvanceFromOutcome(o)
	if err != nil {
		t.Fatalf("AdvanceFromOutcome returned err: %v", err)
	}
	if tr.Renchan {
		t.Errorf("non-dealer ron Renchan = true, want false")
	}
	if m.HandIndex() != 1 {
		t.Errorf("HandIndex after non-dealer ron = %d, want 1", m.HandIndex())
	}
	if m.Dealer() != SeatSouth {
		t.Errorf("Dealer after rotation = %d, want SeatSouth", m.Dealer())
	}
	if m.Honba() != 0 {
		t.Errorf("Honba after rotation = %d, want 0", m.Honba())
	}
	if m.Scores()[SeatSouth] != 26000 || m.Scores()[SeatEast] != 24000 {
		t.Errorf("scores after non-dealer ron = %v, want SeatSouth=26000 SeatEast=24000",
			m.Scores())
	}
}

func TestAdvanceFromOutcomeDealerTsumoRenchan(t *testing.T) {
	m := NewMatch(7)
	// Dealer tsumo: any value works. Use 30fu 2han: base = 30*16 = 480; dealer-tsumo each = roundUp100(480*2) = 1000; total = 3000.
	o := OutcomeTsumo{Winner: SeatEast, Result: mockResult(3000, 480)}

	tr, err := m.AdvanceFromOutcome(o)
	if err != nil {
		t.Fatalf("AdvanceFromOutcome returned err: %v", err)
	}
	if !tr.Renchan {
		t.Errorf("dealer tsumo Renchan = false, want true")
	}
	if m.HandIndex() != 0 {
		t.Errorf("HandIndex after dealer tsumo = %d, want 0 (renchan)", m.HandIndex())
	}
	if m.Dealer() != SeatEast {
		t.Errorf("Dealer after renchan = %d, want SeatEast unchanged", m.Dealer())
	}
	if m.Honba() != 1 {
		t.Errorf("Honba after dealer tsumo = %d, want 1", m.Honba())
	}
}

func TestAdvanceFromOutcomeDealerTenpaiRyuukyokuRenchan(t *testing.T) {
	m := NewMatch(7)
	o := OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatNorth}}

	tr, err := m.AdvanceFromOutcome(o)
	if err != nil {
		t.Fatalf("AdvanceFromOutcome returned err: %v", err)
	}
	if !tr.Renchan {
		t.Errorf("dealer-tenpai ryuukyoku Renchan = false, want true")
	}
	if m.Honba() != 1 {
		t.Errorf("Honba after dealer-tenpai ryuukyoku = %d, want 1", m.Honba())
	}
	if m.Dealer() != SeatEast {
		t.Errorf("Dealer after dealer-tenpai ryuukyoku = %d, want SeatEast unchanged", m.Dealer())
	}
}

func TestAdvanceFromOutcomeRoundWindTransitions(t *testing.T) {
	m := NewMatch(7)
	// Drive 4 non-renchan hands: each is non-dealer ron from the seat to the dealer's right.
	for i := range 4 {
		dealer := m.Dealer()
		nonDealer := dealer.Next()
		o := OutcomeRon{Winner: nonDealer, Loser: dealer, Result: mockResult(1000, 240)}
		if _, err := m.AdvanceFromOutcome(o); err != nil {
			t.Fatalf("AdvanceFromOutcome iter=%d returned err: %v", i, err)
		}
	}
	if m.HandIndex() != 4 {
		t.Errorf("after 4 rotations HandIndex = %d, want 4", m.HandIndex())
	}
	if m.RoundWind() != tile.SouthWind {
		t.Errorf("after East 4 RoundWind = %d, want SouthWind", m.RoundWind())
	}
	if m.Dealer() != SeatEast {
		t.Errorf("after full rotation Dealer = %d, want SeatEast (full circle)", m.Dealer())
	}
}

func TestMatchEndsOnHanchanCompletion(t *testing.T) {
	m := NewMatch(7)
	m.SetTestHandIndex(7) // South 4
	dealer := m.Dealer()
	nonDealer := dealer.Next()
	o := OutcomeRon{Winner: nonDealer, Loser: dealer, Result: mockResult(1000, 240)}

	tr, err := m.AdvanceFromOutcome(o)
	if err != nil {
		t.Fatalf("AdvanceFromOutcome at South 4 returned err: %v", err)
	}
	if !m.IsFinished() {
		t.Errorf("IsFinished after South 4 non-renchan = false, want true")
	}
	if m.FinalOutcome() == nil || m.FinalOutcome().Reason != "hanchan-complete" {
		t.Errorf("FinalOutcome = %+v, want Reason=hanchan-complete", m.FinalOutcome())
	}
	if tr.MatchOutcome == nil {
		t.Errorf("TransitionResult.MatchOutcome is nil, want populated")
	}
}

func TestMatchEndsOnTobi(t *testing.T) {
	m := NewMatch(7)
	m.SetTestScore(SeatNorth, 1500)
	// Plant a haneman dealer ron from East against North.
	// Base = 3000, total = roundUp100(3000*6) = 18000. Plus honba=0. North pays 18000, lands at -16500.
	o := OutcomeRon{Winner: SeatEast, Loser: SeatNorth, Result: mockResult(18000, 3000)}

	_, err := m.AdvanceFromOutcome(o)
	if err != nil {
		t.Fatalf("AdvanceFromOutcome returned err: %v", err)
	}
	if !m.IsFinished() {
		t.Errorf("IsFinished after tobi = false, want true")
	}
	if m.FinalOutcome().Reason != "tobi" {
		t.Errorf("FinalOutcome.Reason = %q, want tobi", m.FinalOutcome().Reason)
	}
	if m.FinalOutcome().BustSeat != SeatNorth {
		t.Errorf("FinalOutcome.BustSeat = %d, want SeatNorth", m.FinalOutcome().BustSeat)
	}
}

func TestAdvanceFromOutcomeOnFinishedMatchReturnsError(t *testing.T) {
	m := NewMatch(7)
	m.SetTestScore(SeatNorth, 1500)
	o := OutcomeRon{Winner: SeatEast, Loser: SeatNorth, Result: mockResult(18000, 3000)}
	if _, err := m.AdvanceFromOutcome(o); err != nil {
		t.Fatalf("first AdvanceFromOutcome returned err: %v", err)
	}
	if !m.IsFinished() {
		t.Fatalf("precondition: match should be finished after tobi")
	}

	scoresBefore := m.Scores()
	_, err := m.AdvanceFromOutcome(o)
	if !errors.Is(err, ErrMatchAlreadyFinished) {
		t.Errorf("second AdvanceFromOutcome err = %v, want ErrMatchAlreadyFinished", err)
	}
	if m.Scores() != scoresBefore {
		t.Errorf("scores mutated by AdvanceFromOutcome on finished match")
	}
}

func countRedTiles(g *Game) int {
	n := 0
	for _, t := range g.wall.allTiles() {
		if t.Red {
			n++
		}
	}
	return n
}

func TestNewMatchDefaultsToAkadoraOn(t *testing.T) {
	m := NewMatch(7)
	if got := countRedTiles(m.CurrentGame()); got != 3 {
		t.Errorf("NewMatch wall red-tile count = %d, want 3 (one per five-rank suit)", got)
	}
}

func TestNewMatchWithOptionsAkadoraOffPropagatesToEveryHand(t *testing.T) {
	m := NewMatchWithOptions(7, MatchOptions{Akadora: false})
	if got := countRedTiles(m.CurrentGame()); got != 0 {
		t.Errorf("NewMatchWithOptions(akadora-off) hand 0 red count = %d, want 0", got)
	}

	for i := 1; i <= 3; i++ {
		o := OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: mockResult(1000, 240)}
		if _, err := m.AdvanceFromOutcome(o); err != nil {
			t.Fatalf("AdvanceFromOutcome iteration %d returned err: %v", i, err)
		}
		if m.IsFinished() {
			t.Fatalf("match ended unexpectedly at iteration %d", i)
		}
		if got := countRedTiles(m.CurrentGame()); got != 0 {
			t.Errorf("hand %d red count = %d, want 0", i, got)
		}
	}
}
