## Why

Today the engine plays a single hand and then halts at `StateRoundOver` — the score struct is bookkept (initialized 25000, riichi deposits deducted) but agari payouts are never applied, the dealer never rotates, and there's no concept of an East/South round or a hanchan. The game ends after one hand. To make the project actually playable as a riichi match, we need a hanchan (East 1-4 + South 1-4) wrapper that applies score payouts, rotates the dealer, tracks honba and riichi-stick carryover, and renders the per-hand context (round/hand/honba/scores) in the TUI.

## What Changes

- New `Match` type in `internal/game` that owns hanchan-level state: scores (per-seat), current dealer seat, current round wind, hand index 0..7 (East 1 → South 4), honba counter, pooled riichi sticks, and the active per-hand `*Game`.
- Score-payout application: when a hand ends in `StateRoundOver`, `Match.AdvanceFromOutcome(Outcome) (TransitionResult, error)` reads the outcome's `*calc.Result.Award`, computes the per-seat deltas (including honba and riichi-stick bonuses), applies them to `Match.scores`, and prepares the next hand.
- Per-hand seat-wind override: dealer is no longer hard-pinned to `SeatEast`. `Match.SeatWindFor(seat) uint8` returns each seat's hand-relative wind (dealer = East, dealer+1 = South, ...), and `Game` queries this getter via a new constructor parameter so `contextForWin` reports the correct yakuhai/round-wind context for non-zero hands.
- Renchan rule: when the dealer wins (tsumo or ron) OR is tenpai at exhaustive ryuukyoku, the same hand index is replayed with `honba++`. Otherwise, dealer rotates (East → South → West → North), honba resets to 0, and `handIndex++`. After hand index 3 the round wind transitions from East to South.
- Riichi-stick carryover: the next agari winner sweeps the entire pool. Sticks remain pooled across renchan and rotation when no agari occurs.
- Game-end conditions: hanchan completes when handIndex would advance past South 4 (handIndex == 8 with no renchan). Tobi (any seat below 0) ends the match immediately. The engine returns a `MatchOutcome` describing the final scores and reason.
- TUI integration: `play.Model` switches from `*game.Game` to `*game.Match`, the status bar renders the live round/hand/honba/riichi-pool/scores, and on `StateRoundOver` the model presents an end-of-hand acknowledgement (any key advances) before transitioning to the next hand. On `MatchOutcomeFinished` the model shows a final standings screen.

## Capabilities

### New Capabilities

- `match-flow`: hanchan match orchestration — score-keeping, dealer rotation, renchan, honba/riichi-pool tracking, end-of-match detection.

### Modified Capabilities

- `game-loop`: dealer is no longer hard-pinned to `SeatEast`; per-hand seat-wind lookup is parameterized via the owning `Match`. The `Game.contextForWin` continues to populate yaku context but reads seat winds through the match-aware getter.
- `play-screen`: the play screen is now bound to a `*game.Match` rather than a single `*game.Game`. Status bar reflects live match state; an end-of-hand confirmation step advances to the next hand; an end-of-match standings screen appears on hanchan completion or tobi.

## Impact

- New: `internal/game/match.go` (Match struct + AdvanceFromOutcome), `internal/game/match_test.go`, `internal/game/payout.go` (per-seat delta computation), `internal/game/payout_test.go`.
- Modified: `internal/game/turn.go` (dealer-aware constructor `NewWithDealer(seed, dealer Seat, roundWind uint8)`, `Game.SeatWindFor(seat)` helper, contextForWin reads seat wind through helper), `internal/game/state.go` (no-op cleanup if needed), `internal/play/play.go` (Model holds `*Match`, status renderer reads Match state, end-of-hand and end-of-match transitions), `internal/play/play_test.go` (multi-hand flow tests), `cmd/play.go` (constructs a Match instead of a Game).
- Removed: none (legacy `New(seed)` constructor remains for the standalone-hand `mahjong calc` paths and existing engine tests).
