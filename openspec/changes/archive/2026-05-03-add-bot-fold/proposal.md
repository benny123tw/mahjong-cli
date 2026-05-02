## Why

Danger-aware discards landed in `add-bot-defense` (genbutsu/suji penalty K=2000), but bots still always push. A bot at shanten=2 facing an opponent's declared riichi will pick the tile with the best isolation+danger blend — which can still favor a less-isolated unsafe tile over a perfectly-genbutsu tile when the isolation gap is bigger than the K=2000 penalty headroom. A real riichi player folds in that situation: when winning is hopeless, you discard the SAFEST tile, full stop. This change adds that fold step so non-riichi bots stop dealing into riichi declarers when they have no realistic path to tenpai.

## What Changes

- A new `(b *Bot) FoldDiscard(hand []tile.Tile, danger map[uint8]int) int` SHALL be added in `internal/game/bot.go`. It uses the same isolation+danger blend as `DangerAwarePickDiscard` but with `dangerPenaltyKFold = 1_000_000` (effectively infinite vs. the ~1000-range isolation score) so danger ALWAYS dominates isolation. When the danger map is empty the function SHALL fall back to `PickDiscard` (graceful degradation, never panics), matching the `DangerAwarePickDiscard` contract.
- The play package SHALL decide which discard function to call. In `dispatchBotDiscard` (`internal/play/play.go`), when the bot's shanten on its current 14-tile hand is >= 2 AND the assembled `danger` map is non-empty (i.e., at least one opponent has declared riichi), the dispatcher SHALL call `bot.FoldDiscard` instead of `bot.DangerAwarePickDiscard`. The shanten check uses `hand.Shanten(hand.Hand{Concealed: bot's 14-tile hand})` — when shanten == 0 or 1 the bot is close to tenpai and continues to push via the existing `DangerAwarePickDiscard`.
- Fold mode only affects the discard choice. The bot is NOT prohibited from declaring tsumo or ron during fold mode; if a winning tile lands, the bot still wins. Riichi declaration requires shanten==0 by definition, so the fold gate (shanten >= 2) excludes it automatically.
- New unit tests in `internal/game/bot_test.go` SHALL cover: a bot with 14-tile hand at shanten=2 facing a danger map containing one genbutsu tile (danger=0) and one suji-safe tile (danger=1) plus several unknown-danger tiles (danger=2 default); FoldDiscard SHALL pick the genbutsu tile even when an unknown-danger tile has a much higher isolation score.

## Non-Goals

- Pre-riichi danger reading — a steep, fast-discard pond from a non-riichi opponent is still treated as zero-danger by this change. Yomi (opponent hand reconstruction) and kabe (wall reading) are also out of scope.
- Partial-fold / semi-defensive play — the change is binary: full fold when shanten >= 2 + riichi, otherwise full push (with the existing K=2000 danger blend). Mid-shanten "duck while improving" is a future heuristic.
- Calls (pon / chi / kan) by a folding bot — `ShouldPon`, `ShouldChi`, and the kan-call logic are unchanged. A folding bot SHOULD also stop calling, but that's a separate change so this one stays narrowly scoped.
- Multi-riichi with conflicting safety — when two opponents declare riichi simultaneously, the existing `assembleDangerMap` already merges via min-aggregation; this change inherits that behaviour without changes.
- A new fold-eligibility threshold — shanten >= 2 is the fixed gate; no configuration knob.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `game-loop`: `Bot Decision Strategy` (or whatever the existing requirement covering bot discard scoring is named) updated to add fold-mode behaviour: when shanten >= 2 AND any opponent has declared riichi, the bot SHALL pick the safest tile by danger map regardless of isolation score.

## Impact

- Affected specs: `game-loop` (modified — bot discard strategy gains fold-mode rule)
- Affected code:
  - Modified: `internal/game/bot.go` (new `FoldDiscard` method + `dangerPenaltyKFold` constant), `internal/game/bot_test.go` (fold-mode tests), `internal/play/play.go` (`dispatchBotDiscard` chooses between `DangerAwarePickDiscard` and `FoldDiscard` based on shanten + danger-map presence)
  - New: none
  - Removed: none
