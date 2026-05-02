## Context

The current engine treats one hand as a complete game. `game.New(seed)` shuffles a wall, deals 13 tiles to each seat, sets state to `AwaitingDraw{SeatEast}`, and runs the per-hand state machine. When the round ends, `g.state` becomes `StateRoundOver` and that's the end of the road. `g.scores` tracks per-seat point totals — initialized to 25000, decremented by 1000 on riichi declarations — but never receives agari payouts. The play package wraps a single `*game.Game` and never recreates it.

Riichi seat winds are computed by a hard-coded `Seat.SeatWind() = EastWind + uint8(seat)` mapping. That works for hand 1 (East dealership stays at SeatEast) but breaks the moment dealer rotates: in East 2 the dealer should be SeatSouth, who must be treated as East-wind for yakuhai/round-wind purposes.

The TUI's status row currently hardcodes `East 1 · Honba 0 · Score 25000`. It reads `m.game.Wall().LiveRemaining()` and `m.game.DoraIndicators()` but ignores hand index and honba.

To make this a playable hanchan, we need a layer above `Game` that owns persistent match state and rebuilds a fresh `*Game` per hand with the right dealer, seed, and seat-wind context.

## Goals / Non-Goals

**Goals:**

- Hanchan match: 8 hands (East 1-4, South 1-4) playable end-to-end with score tracking.
- Score payout application: `OutcomeRon`, `OutcomeTsumo`, and `OutcomeRyuukyoku` each translate to per-seat deltas that mutate `Match.scores`.
- Dealer rotation with renchan: dealer wins or dealer-tenpai-at-ryuukyoku → same hand, honba++. Otherwise rotate (East seat → next physical seat) and reset honba.
- Round-wind transition: hands 0-3 are East round, hands 4-7 are South round.
- Per-hand seat-wind override: each `Game` knows its dealer seat; `SeatWindFor` returns dealer-relative winds.
- Riichi-stick carryover: pool sticks across renchans/rotations with no agari; agari sweeps the entire pool.
- Match-end: either hanchan completes (handIndex would advance past 7 with no renchan) or tobi (any seat below 0 → immediate end).
- TUI integration: status row reflects live match state; end-of-hand acknowledgement step; end-of-match standings.

**Non-Goals:**

- West round and beyond (only East and South — standard hanchan).
- Tonpuusen mode (East-only 4-hand match).
- Uma / oka end-of-match score adjustments (just raw point totals).
- Sudden-death extensions when no leader at South 4 end (just stop at South 4).
- Match save/restore.
- Honba bonus calculations beyond the standard +100 per honba per non-winning seat (no abadon honba caps, no rule variations).
- Renchan limit (some rule sets cap dealer renchan at e.g. honba 5; we don't enforce a cap).
- Multi-ron / triple-ron: kept as v1 (resolver picks first by priority order); the match-level layer doesn't change ron-priority semantics.

## Decisions

### `Match` Owns Persistent State; `Game` Stays Round-Scoped

The `Match` struct lives in `internal/game/match.go` and holds: `scores [4]int`, `dealer Seat`, `roundWind uint8`, `handIndex int`, `honba int`, `riichiSticks int`, `seed int64` (base), and `currentGame *Game` (the active hand's per-round state).

The existing `Game` keeps its current shape but gains a constructor variant `NewWithDealer(seed int64, dealer Seat, roundWind uint8) *Game` that wires the dealer and round wind through to per-game state. The legacy `New(seed)` constructor delegates to `NewWithDealer(seed, SeatEast, tile.EastWind)` — backwards-compatible for existing tests and the `mahjong calc` CLI path.

`Match.AdvanceFromOutcome(Outcome) (TransitionResult, error)` is the workhorse: read the outcome, compute payouts, apply to `Match.scores`, decide renchan vs rotate, build the next `*Game` (or signal match-over), return a `TransitionResult` describing what happened (so the TUI can render an ack screen).

**Alternative considered:** Refactor `Game` itself to span the whole match. Rejected — the per-hand state machine (wall, hands, discards, melds, dora indicators, ippatsu windows, etc.) all reset every hand. Mixing match-scope and hand-scope state in one struct fights the natural boundary; a wrapper is cleaner.

### Per-Hand Seat Wind Computed From Dealer Offset

Add `Match.SeatWindFor(seat Seat) uint8`: the dealer is East-wind, dealer.Next() is South-wind, and so on. Computed as `tile.EastWind + uint8((seat - dealer + 4) % 4)`.

The per-hand `Game` gains a stored `dealer Seat` field and exposes `Game.SeatWindFor(seat)` that wraps the same calculation. `contextForWin` switches from `winner.SeatWind()` to `g.SeatWindFor(winner)`. Bot dispatcher (`play.botContextForWin`) similarly migrates to `m.game.SeatWindFor(seat)`.

The hard-coded `Seat.SeatWind()` method stays on the `Seat` type — but only as a default for the `mahjong calc` CLI path, where the user supplies the seat wind directly anyway. Engine code must call `Game.SeatWindFor` instead.

**Alternative considered:** Rotate the players themselves (always make `SeatEast` the dealer; the human cycles through different physical seats). Rejected — that breaks every test that hardcodes `HumanSeat = SeatSouth` and complicates TUI rendering (the human would have to be drawn at different screen positions). Keeping physical seats fixed and computing winds from offset is far less invasive.

### Score Payout Lives In `internal/game/payout.go`

A pure function `ComputePayouts(outcome Outcome, ctx PayoutContext) [4]int` returns per-seat deltas (positive for the winner, negative for losers). `PayoutContext` carries the dealer seat (so dealer-double payouts work for ryuukyoku and tsumo distribution), honba, and riichi-pool size.

The function lives in its own file because the payout rules are non-trivial and warrant focused tests independent of `Match` state machine plumbing:
- **Ron**: winner gets `Award.Total` from loser, plus `100 × honba × 3` from loser, plus the entire riichi-stick pool. Losers' deltas: only the discarder pays.
- **Tsumo**: payout split derived from base score. Non-dealer tsumo: each non-dealer pays `roundUp100(base * 1)`, dealer pays `roundUp100(base * 2)`; honba adds `100 × honba` per payer (3 payers). Dealer tsumo: each non-dealer pays `roundUp100(base * 2)`; honba adds `100 × honba` per payer.
- **Ryuukyoku**: dealer-tenpai counts trigger no special payout to dealer (only the standard noten payment). Tenpai/noten transfer: 3000 total flows from noten seats to tenpai seats. 1 tenpai → 3000 from each noten (3 × 1000 = 3000 in, 1000 out per noten). 2 tenpai → 1500 each from 2 noten (each noten pays 1500). 3 tenpai → 1000 each from the 1 noten (noten pays 3000 total). 0 or 4 tenpai → no transfer. Riichi sticks stay pooled.

`Award.Total` already encodes the gross payout for tsumo dealer (`each * 3`) and tsumo non-dealer (`nonDealerPay*2 + dealerPay`). The payout function re-derives the per-seat split from `Award.Base` rather than re-computing — `Award.Base` is the source of truth.

**Alternative considered:** Apply payouts inside `Match.AdvanceFromOutcome` directly. Rejected — payout logic has many cases (ron-vs-tsumo, dealer-vs-non-dealer, ryuukyoku transfer matrix) that benefit from being unit-testable in isolation. Splitting it out also makes the logic easy to extend with red-five / dora-bonus tweaks later.

### Renchan Detection Uses Outcome Inspection, Not Engine Hooks

`Match.AdvanceFromOutcome` inspects the outcome to detect renchan:

| Outcome                           | Dealer Renchan? |
| --------------------------------- | --------------- |
| `OutcomeRon{Winner == dealer}`    | Yes             |
| `OutcomeRon{Winner != dealer}`    | No              |
| `OutcomeTsumo{Winner == dealer}`  | Yes             |
| `OutcomeTsumo{Winner != dealer}`  | No              |
| `OutcomeRyuukyoku{TenpaiPlayers contains dealer}` | Yes |
| `OutcomeRyuukyoku{!contains dealer}`              | No  |

Renchan: same hand index, `honba++`, riichi pool unchanged.
Rotate: `handIndex++`, dealer = `dealer.Next()`, `honba = 0`, riichi pool unchanged (carries to whoever wins next).
Round-wind transition: when `handIndex` advances from 3 to 4, switch `roundWind` from `tile.EastWind` to `tile.SouthWind`.

After processing the transition, build the next per-hand `*Game` via `NewWithDealer(seed + handIndex, dealer, roundWind)` — seeded deterministically per hand for replay reproducibility.

**Alternative considered:** Introduce an explicit `RenchanContext` field on outcomes or have the engine emit a renchan signal. Rejected — outcomes already carry the winner seat (or tenpai list); inspecting them in the match layer keeps the engine surface small.

### TUI Adds an End-of-Hand Acknowledgement Step

After `StateRoundOver` and `Match.AdvanceFromOutcome`, the TUI displays a summary panel showing the outcome, score deltas, and updated totals. The panel waits for any keypress before triggering the next hand's setup. This is a deliberate UX pause — without it, the screen flips through ron/tsumo events too fast to read.

Implementation: `play.Model` gains a `pendingTransition *MatchTransitionAck` field. When `m.game.State()` (now `m.match.CurrentGame().State()`) is `StateRoundOver`, the model invokes `Match.AdvanceFromOutcome`, stores the resulting `MatchTransitionAck` in `pendingTransition`, and renders the ack panel. Any `tea.KeyPressMsg` while `pendingTransition != nil` clears it, switches to the next `*Game`, and resumes normal play (or shows the final standings if `MatchOutcomeFinished`).

**Alternative considered:** Auto-advance with a 2-second delay. Rejected — slows things down by always taking 2s, regardless of how engaged the player is.

### Tobi (Bust) Ends the Match Immediately

After applying payouts, `Match.AdvanceFromOutcome` checks `for s := range Seat(4) { if scores[s] < 0 { return MatchOutcomeFinished{Reason: "tobi", BustSeat: s} } }`. The TUI surfaces the reason in the standings screen.

**Alternative considered:** Allow negative scores. Rejected — standard riichi rule is tobi-end. Real-world play sometimes uses 0 instead of negative as the threshold; we use strict `< 0` for simplicity.

## Risks / Trade-offs

[Risk: changing `Game.contextForWin` to use `SeatWindFor` could regress single-hand tests that assume `SeatEast.SeatWind() == EastWind`] → Mitigation: the legacy `New(seed)` constructor pins dealer to `SeatEast`, so `SeatWindFor(SeatEast) == EastWind` matches the old behavior exactly. Tests that don't construct via `NewWithDealer` see no observable change.

[Risk: payout computation has many branches; off-by-one errors in dealer-vs-non-dealer rounding accumulate over a hanchan and produce wrong final scores] → Mitigation: unit-test every payout case in `payout_test.go` against textbook values (e.g., 30fu 3han non-dealer ron = 3900 + honba; dealer tsumo mangan = 4000 all). Cross-reference with `score.Compute` outputs.

[Risk: per-hand RNG seeding (`seed + handIndex`) means the seed-7 game-2 is not the seed-9 game-0 — anyone debugging a specific seed must also know which hand they're looking at] → Mitigation: log the per-hand effective seed in `Match.advanceTo(handIndex)`. The `seed=7,hand=2` form is the canonical replay key.

[Risk: end-of-hand ack screen blocks bot ticks; if the player walks away mid-ack, the match stays paused indefinitely] → Mitigation: acceptable — single-player CLI, no time pressure. A future change can add an auto-advance timeout if needed.

[Risk: the dealer-wind refactor touches every yaku-context-related code path; missing a callsite produces silently-wrong yakuhai detection in later hands (e.g., the East 2 dealer's South-wind tile is mis-detected as round-wind in a South round)] → Mitigation: grep for every use of `seat.SeatWind()` and `SeatEast.SeatWind()` in the engine; route them through `g.SeatWindFor(seat)` or `m.SeatWindFor(seat)`. Add an integration test that runs hanchan to South 1 and validates a non-default seat-wind yakuhai trigger.
