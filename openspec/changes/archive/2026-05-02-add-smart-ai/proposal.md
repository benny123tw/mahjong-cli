## Why

Bots currently auto-pass every claim window (`InputResolveClaims{Claims: nil}`) and never declare tsumo even when their 14-tile hand wins. Their `ShouldRiichi` returns false unconditionally. The result is a one-sided game: the human can play perfectly defensively because no opponent will ever pon, chi, ron, or tsumo. Live games against this bot feel like solo solitaire.

The mechanics are already 80% built: the engine accepts `ClaimRon` / `ClaimPon` / `ClaimChi`, `calc.Analyze` returns scoring results for any seat, `Bot.ShouldPon` and `Bot.ShouldChi` already encode the spec'd heuristics, and add-human-agari just plumbed riichi state for the human path. What's missing is the wiring in `handleBotTick` plus a riichi-declaration heuristic for bots and a generalization of the furiten gate from human-only to all-seats.

## What Changes

In `handleBotTick`'s `AwaitingDiscard` branch, before submitting a discard the bot SHALL check whether its 14-tile hand wins yakufully via `calc.Analyze`; if so, it submits `InputDeclareTsumo` and skips the discard. Otherwise the bot evaluates riichi: if `ShouldRiichi` returns true (new heuristic — see below), it picks the first tile index whose remaining 13-tile hand has shanten=0 and submits `InputDiscard{Index: idx, Riichi: true}`. Otherwise it falls back to the existing isolation-heuristic discard.

In `handleBotTick`'s `AwaitingClaims` branch, instead of unconditionally passing, the engine SHALL collect a `Claim` from each non-discarder bot. For each bot: first check ron — `calc.Analyze` on `concealed + discard` returns non-nil AND `Game.IsFuriten(seat)` returns false → `ClaimRon`. Otherwise check pon — `Bot.ShouldPon` returns true → `ClaimPon`. Otherwise check chi (only kamicha) — `Bot.ShouldChi` returns the first option → `ClaimChi{ChiTiles: option}`. Otherwise pass. The collected claims map is submitted via a single `InputResolveClaims`; the existing resolver handles priority (ron > pon > chi).

`Bot.ShouldRiichi` SHALL be reimplemented from "always false" to: declare when ALL hold — post-discard hand is tenpai (some discardable index leaves shanten=0), the seat is concealed, the seat's score is ≥1000, and `Wall.LiveRemaining() >= 4`. The returned tile-choice index is the first scanned position (0..len-1) whose post-discard 13-tile hand satisfies tenpai. This is a deliberately simple heuristic — better tile-choice strategy (machi-quality scoring, dora retention) is out of scope.

`Game.stepFromAwaitingClaims`'s `ClaimRon` branch currently gates furiten only when `winner == HumanSeat`. With bots now ronning, the gate SHALL apply to all seats: any seat in permanent furiten (machi tile in own pond) cannot ron. The same `Game.IsFuriten` query already handles per-seat checks correctly.

The TUI's `BotTickMsg` cadence (250ms) stays unchanged. After a bot tsumo or ron, the round transitions to `RoundOver` and the bot tick stops naturally.

## Non-Goals

- Danger awareness / hand-reading defense (a bot deciding to fold safe tiles when an opponent declared riichi). Smart defense ships in a follow-up `add-bot-defense`.
- Bot riichi tile-choice strategy beyond "first scanned tenpai". A real bot would compare wait-quality (number of waits, wait-tiles-remaining, dora retention) across all tenpai-leaving discards. Out of scope.
- Pao / sekinin-barai liability rules.
- Bot kan calls (deferred to `add-kan-support`).
- Bot temporary furiten (passing on an opponent's machi tile makes you furiten until your next draw). Requires per-seat machi-tile-passed tracking — deferred. Bots only respect permanent furiten.
- Bot ippatsu / double-riichi awareness in heuristics. The engine sets the flags correctly when bots declare riichi; the heuristic doesn't optimize for them.
- Multi-bot calls on the same discard with non-trivial priority adjudication beyond what `ResolveClaims` already handles. Existing resolver suffices.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `game-loop`: extends `Bot Decision Strategy` to flip the Riichi rule from "Never" to a tenpai+funds+wall heuristic with a tile-choice algorithm. Adds the bot tsumo and bot ron paths to the bot-decision contract. Generalizes the ron furiten gate from human-only to all seats.
- `play-screen`: extends bot-action dispatch (currently auto-pass + isolation discard) to wire the new bot decisions through `InputDeclareTsumo`, `InputResolveClaims{ClaimRon}`, `InputResolveClaims{ClaimPon}`, `InputResolveClaims{ClaimChi}`, and `InputDiscard{Riichi: true}`.

## Impact

- Affected specs: modified capability `game-loop` (Bot Decision Strategy + Human Ron From Claim Window → Ron From Claim Window); modified capability `play-screen` (bot dispatch in handleBotTick).
- Affected code:
  - Modified:
    - internal/game/bot.go (ShouldRiichi reimplementation; possibly add ShouldTsumo helper, ShouldRon helper)
    - internal/game/turn.go (furiten gate in ClaimRon branch generalized: drop the `winner == HumanSeat` guard)
    - internal/play/play.go (handleBotTick's three branches: AwaitingDiscard adds tsumo + riichi check, AwaitingClaims iterates bots + calls)
    - internal/game/bot_test.go (extend ShouldRiichi tests, add ShouldTsumo / ShouldRon tests)
    - internal/play/play_test.go (extend bot-tick tests with tsumo + ron scenarios)
  - New:
    - (none — fits into existing files)
  - Removed:
    - (none)
