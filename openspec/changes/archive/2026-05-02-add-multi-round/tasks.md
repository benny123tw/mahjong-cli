## 1. Engine: Per-Hand Dealer-Relative Seat Wind (decision: per-hand seat wind computed from dealer offset)

- [x] 1.1 Update Per-Hand Dealer-Relative Seat Wind — in `internal/game/turn.go`, add `dealer Seat` and (already-present) `roundWind uint8` fields to `Game` initialized via a new `NewWithDealer(seed int64, dealer Seat, roundWind uint8) *Game` constructor. Refactor `New(seed int64) *Game` to delegate to `NewWithDealer(seed, SeatEast, tile.EastWind)` so existing tests see no behavioral change.
- [x] 1.2 Add `Game.SeatWindFor(seat Seat) uint8` returning `tile.EastWind + uint8((seat - g.dealer + 4) % 4)`. Document that this replaces direct `Seat.SeatWind()` calls on the engine path.
- [x] 1.3 In `Game.contextForWin`, change `winner.SeatWind()` to `g.SeatWindFor(winner)`. The same refactor applies anywhere else in `internal/game/*.go` that reads `seat.SeatWind()` for a winning-context purpose (grep for `.SeatWind()` and patch each engine-internal call).
- [x] 1.4 In `internal/play/play.go`'s `botContextForWin`, replace `seat.SeatWind()` with `m.game.SeatWindFor(seat)` so bot win evaluations also see hand-relative winds. The `RoundWind` field continues to use `m.game.RoundWind()` (already correct).
- [x] 1.5 Add `internal/game/turn_test.go::TestSeatWindForDealerRelative` covering both `NewWithDealer(7, SeatEast, ...)` (East-wind for East) and `NewWithDealer(7, SeatSouth, ...)` (East-wind for South, North-wind for East).
- [x] 1.6 Add `internal/game/turn_test.go::TestContextForWinUsesSeatWindFor` planting a winning hand at `SeatNorth` in a `NewWithDealer(seed, SeatSouth, ...)` game and asserting `contextForWin(SeatNorth, true).SeatWind == tile.WestWind` (not `tile.NorthWind`).

## 2. Engine: ComputePayouts (decision: score payout lives in `internal/game/payout.go`)

- [x] 2.1 Update Score Payout Computation — create `internal/game/payout.go` with `type PayoutContext struct { Dealer Seat; Honba int; RiichiSticks int }` and `ComputePayouts(o Outcome, ctx PayoutContext) [4]int`.
- [x] 2.2 Implement the ron branch: `case OutcomeRon`, compute `winnerGain := o.Result.Award.Total + 300*ctx.Honba + 1000*ctx.RiichiSticks`; `loserPay := o.Result.Award.Total + 300*ctx.Honba`. Return `[4]int{}` with `[winner] = +winnerGain, [loser] = -loserPay`.
- [x] 2.3 Implement the tsumo branch: `case OutcomeTsumo`, derive per-payer amounts from `o.Result.Award.Base`. If winner is dealer (`o.Winner == ctx.Dealer`): each non-dealer pays `roundUp100(Base*2) + 100*ctx.Honba`. If non-dealer winner: each non-dealer (other than the winner) pays `roundUp100(Base*1) + 100*ctx.Honba`, and the dealer pays `roundUp100(Base*2) + 100*ctx.Honba`. Winner gains the sum of payments + `1000*ctx.RiichiSticks`.
- [x] 2.4 Implement the ryuukyoku branch: `case OutcomeRyuukyoku`, count tenpai. If 0 or 4 tenpai → return all zeros. Otherwise: noten count = 4 - tenpai count, total transfer = 3000, each tenpai gains `3000/tenpaiCount`, each noten loses `3000/notenCount`. Honba does NOT add to ryuukyoku.
- [x] 2.5 Add a private `roundUp100(n int) int` helper (or copy from `internal/riichi/score/score.go`); justify the duplication in a one-line comment as keeping payout self-contained for testability.
- [x] 2.6 Create `internal/game/payout_test.go` with one test per row of the spec's payout table: `TestComputePayoutsNonDealerRon30Fu3HanWithHonba2`, `TestComputePayoutsDealerTsumoMangan`, `TestComputePayoutsRyuukyokuOneTenpai`, `TestComputePayoutsRyuukyokuAllTenpai`, `TestComputePayoutsRyuukyokuAllNoten`, `TestComputePayoutsRiichiStickSweep`. Each test constructs an Outcome with explicit `*calc.Result` (or mocked Award), invokes `ComputePayouts`, and asserts the per-seat deltas against the textbook values from the spec.

## 3. Engine: Match struct (decision: `Match` owns persistent state; `Game` stays round-scoped)

- [x] 3.1 Update Hanchan Match Structure — create `internal/game/match.go` with `type Match struct { scores [4]int; dealer Seat; roundWind uint8; handIndex int; honba int; riichiSticks int; seed int64; currentGame *Game; outcome *MatchOutcome }`.
- [x] 3.2 Implement `NewMatch(seed int64) *Match` initializing scores to `[25000]*4`, dealer = `SeatEast`, roundWind = `tile.EastWind`, handIndex = 0, honba = 0, riichiSticks = 0, currentGame = `NewWithDealer(seed, SeatEast, tile.EastWind)`.
- [x] 3.3 Add accessors: `Match.Scores() [4]int` (defensive copy), `Match.Dealer() Seat`, `Match.RoundWind() uint8`, `Match.HandIndex() int`, `Match.Honba() int`, `Match.RiichiSticks() int`, `Match.CurrentGame() *Game`, `Match.IsFinished() bool`, `Match.FinalOutcome() *MatchOutcome`.
- [x] 3.4 Add `Match.HandLabel() string` returning the human-readable hand name: `"East 1"` for handIndex=0, `"East 2"` for handIndex=1, ..., `"South 4"` for handIndex=7. Used by the TUI status bar.

## 4. Engine: Match.AdvanceFromOutcome (decision: renchan detection uses outcome inspection, not engine hooks)

- [x] 4.1 Update Match Advancement From Outcome — implement `Match.AdvanceFromOutcome(o Outcome) (TransitionResult, error)` returning `ErrMatchAlreadyFinished` if `Match.outcome != nil`. Otherwise: build `PayoutContext{Dealer: m.dealer, Honba: m.honba, RiichiSticks: m.riichiSticks}`, call `ComputePayouts(o, ctx)`, apply deltas to `m.scores`. Track riichi-stick movement: ron/tsumo zero out the pool; ryuukyoku leaves it unchanged.
- [x] 4.2 Implement renchan detection per the design's outcome→renchan table: dealer-win on tsumo/ron OR dealer-tenpai-on-ryuukyoku → renchan; otherwise rotate. Use `outcomeWinner(o)` and `outcomeTenpaiContains(o, m.dealer)` helpers (or inline the type switch).
- [x] 4.3 Update Per-Hand Game Construction Determinism — on renchan: `m.honba++`, `m.handIndex` unchanged, dealer unchanged, build the next `*Game` via `NewWithDealer(m.seed + int64(m.handIndex), m.dealer, m.roundWind)` (same handIndex → same per-hand seed → identical wall replay). On rotation: `m.handIndex++`, `m.honba = 0`, `m.dealer = m.dealer.Next()`, then construct the next `*Game` with the bumped handIndex. If `m.handIndex` crosses from 3 to 4, set `m.roundWind = tile.SouthWind`.
- [x] 4.4 Implement Match End Conditions per the decision "tobi (bust) ends the match immediately" — after applying deltas: check tobi (`for s := range Seat(numSeats) { if m.scores[s] < 0 { ... }`) → set `m.outcome = &MatchOutcome{Reason: "tobi", BustSeat: s}`. After advancing handIndex on rotation: if `m.handIndex >= 8` → set `m.outcome = &MatchOutcome{Reason: "hanchan-complete"}`. When match is finished, do NOT construct a new `*Game`; leave `m.currentGame` pointing at the just-finished hand for last-hand inspection. Tobi check fires regardless of whether the outcome was renchan or rotation.
- [x] 4.5 Construct and return the `TransitionResult{Deltas [4]int, NewTotals [4]int, Renchan bool, NewHandIndex int, MatchOutcome *MatchOutcome}`.

## 5. Engine: Match tests

- [x] 5.1 Create `internal/game/match_test.go::TestNewMatchInitialState` asserting all four scores are 25000, dealer is `SeatEast`, handIndex is 0, honba is 0, roundWind is `tile.EastWind`, currentGame is non-nil.
- [x] 5.2 Add `TestAdvanceFromOutcomeNonDealerRonRotates`: construct match, plant outcome `OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: <30fu 1han>}`, call `AdvanceFromOutcome`. Assert renchan = false, handIndex = 1, dealer = `SeatSouth`, honba = 0, scores reflect the 1500-point delta.
- [x] 5.3 Add `TestAdvanceFromOutcomeDealerTsumoRenchan`: plant `OutcomeTsumo{Winner: SeatEast, Result: ...}`. Assert renchan = true, handIndex = 0, dealer = `SeatEast`, honba = 1.
- [x] 5.4 Add `TestAdvanceFromOutcomeDealerTenpaiRyuukyokuRenchan`: plant `OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatNorth}}`. Assert renchan = true, honba = 1, dealer unchanged.
- [x] 5.5 Add `TestAdvanceFromOutcomeRoundWindTransitions`: drive a match through 4 non-renchan rotations to land at handIndex = 4. Assert `m.RoundWind() == tile.SouthWind`, `m.HandIndex() == 4`, `m.Dealer() == SeatEast` (rotated full circle).
- [x] 5.6 Add `TestMatchEndsOnHanchanCompletion`: drive a match to handIndex = 7, plant a non-renchan outcome. Assert `m.IsFinished() == true`, `m.FinalOutcome().Reason == "hanchan-complete"`.
- [x] 5.7 Add `TestMatchEndsOnTobi`: plant a match with `SeatNorth` at score 1500, then plant `OutcomeRon{Winner: SeatEast, Loser: SeatNorth, Result: <haneman 12000>}`. Assert `m.IsFinished() == true`, `m.FinalOutcome().Reason == "tobi"`, `BustSeat == SeatNorth`. (Use a SetTestScore helper added in 3.3 if needed for setup.)
- [x] 5.8 Add `TestAdvanceFromOutcomeOnFinishedMatchReturnsError`: invoke `AdvanceFromOutcome` twice; second call returns `ErrMatchAlreadyFinished` and does not mutate scores. Use a hand-end + tobi setup so the first call finishes the match.

## 6. TUI: Match-Bound Model (decision: TUI adds an end-of-hand acknowledgement step)

- [x] 6.1 Update Match-Bound Model — add `NewWithMatch(renderer Renderer, m *game.Match) Model` to `internal/play/play.go`. Internally store `match *game.Match` (alongside or replacing the existing `game *game.Game` field; deprecate `NewWithGame` if reasonable, otherwise keep it for tests). `Model.GameState()` delegates to `m.match.CurrentGame().State()` when match is non-nil.
- [x] 6.2 Refactor every `m.game.Foo()` accessor used elsewhere in `play.go` to read through `m.match.CurrentGame().Foo()` when a match is bound. Keep the legacy `m.game` path for tests that construct via `NewWithGame`.
- [x] 6.3 Update End-of-Hand Acknowledgement — add `pendingTransition *game.TransitionResult` field to `Model`. When `m.GameState()` returns `StateRoundOver` and `pendingTransition == nil`, the next call to `Update` SHALL invoke `m.match.AdvanceFromOutcome(...)`, store the result in `pendingTransition`, and render the ack panel. The next hand's state machine does NOT begin processing inputs until the player presses any key.
- [x] 6.4 Render the ack panel: a centered box showing the outcome description (e.g., "South ron from East — 1500"), the four per-seat deltas, the new totals, and (on renchan) the new honba count. Replace the normal layout while ack is pending.
- [x] 6.5 In `handleKey`, when `pendingTransition != nil`, any key press SHALL clear the pending transition and (if not finished) resume the play layout with the new hand's state machine.

## 7. TUI: Status Bar and End-of-Match Standings

- [x] 7.1 Update `renderStatus` and `renderCentreInfo` in `internal/play/play.go` to read `m.match.HandLabel()`, `m.match.Honba()`, `m.match.RiichiSticks()`, and per-seat scores from `m.match.Scores()`. Remove the hardcoded "East 1 · Honba 0 · Score 25000" string and the centre's hardcoded "Round: East 1".
- [x] 7.2 Update End-of-Match Standings Screen — when `m.match.IsFinished()`, `View()` SHALL render the standings panel: four rows sorted by score descending (seat name + final score), the reason ("hanchan-complete" or "tobi: <seat>"), and a "[q] Quit" footer. The standings panel replaces the normal layout entirely.
- [x] 7.3 In `handleKey`, when `m.match.IsFinished()`: ignore all keys except `q` and `ctrl+c`, which return `tea.Quit`.

## 8. TUI: Tests for Match Flow

- [x] 8.1 Add `internal/play/play_test.go::TestStatusBarReflectsMatchState`: construct a match, drive it via `SetTestScore`/`SetTestDealer`/`SetTestHandIndex` helpers (added on Match for tests) into a known E2/honba=1/sticks=1 state, render the model, assert the status string contains "East 2", "Honba 1", "Riichi 1", and the planted score numbers.
- [x] 8.2 Add `TestEndOfHandAckPanelOnRoundOver`: construct a match, force `CurrentGame()` into `StateRoundOver{Outcome: OutcomeRon{...}}`, send any tea.Msg, assert `pendingTransition` is non-nil and the rendered View contains "ron" and the per-seat deltas.
- [x] 8.3 Add `TestKeypressOnAckAdvancesToNextHand`: starting from the ack-panel state of 8.2, send a `KeyPressMsg`, assert pendingTransition is nil, the underlying `*Game` is the new East 2 hand, and the rendered View no longer shows the ack panel.
- [x] 8.4 Add `TestStandingsScreenOnHanchanCompletion`: drive a match to a finished state with `Reason = "hanchan-complete"` (use SetTestScore + a forced final-hand outcome). Render View, assert it contains all four seat names with their final scores and the reason string. Send `tea.KeyPressMsg{Code: 'q'}`, assert `tea.Quit` is in the returned cmd chain.

## 9. Wiring: cmd/play

- [x] 9.1 Locate `cmd/play.go` (or wherever `play.NewWithGame` is currently invoked from the CLI). Switch construction to `play.NewWithMatch(renderer, game.NewMatch(seed))`.
- [x] 9.2 Ensure the seed flag (`--seed N`) feeds into `game.NewMatch(seed)` rather than `game.New(seed)`.

## 10. Verification (decision: integration coverage)

- [x] 10.1 Confirm: helper coverage — `TestComputePayouts*` exist in `internal/game/payout_test.go`. Match-state coverage — `TestNewMatchInitialState`, `TestAdvanceFromOutcome*` exist in `internal/game/match_test.go`. Per-hand seat wind coverage — `TestSeatWindForDealerRelative`, `TestContextForWinUsesSeatWindFor` exist in `internal/game/turn_test.go`. TUI coverage — `TestStatusBarReflectsMatchState`, `TestEndOfHandAckPanelOnRoundOver`, `TestKeypressOnAckAdvancesToNextHand`, `TestStandingsScreenOnHanchanCompletion` exist in `internal/play/play_test.go`.
- [x] 10.2 Run `go test ./...` and confirm all suites pass (including the existing single-hand tests that go through `New(seed)`).
- [x] 10.3 Run `golangci-lint run ./...` and confirm 0 issues.
- [x] 10.4 Run `spectra validate add-multi-round` and confirm valid.
