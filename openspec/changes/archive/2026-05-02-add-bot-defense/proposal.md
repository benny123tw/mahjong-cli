## Why

`add-smart-ai` taught bots how to win — tsumo, ron, pon, chi, riichi all wired. But their discard heuristic is still pure isolation: pick the most-disconnected tile from the bot's own hand, ignoring everything on the table. The result is a hilariously exploitable defender: declare riichi as the human, watch every opponent immediately deal you the winning tile.

A second gap: temporary furiten is partially missing. When an opponent discards a bot's machi tile and the bot passes (because the dispatcher chose not to ron — e.g., yakuless wait, dispatcher bug, future "bot judgment to fake-pass"), the bot SHALL be locked out of ron until their next own draw. Today's `IsFuriten` only checks the bot's own pond; the temporary case is unimplemented and would let a bot pass on Tile T then ron on Tile T from a different seat.

These are separate but related: defense AND furiten correctness make the bots feel like they're following the same rules and reading the same board the human is.

## What Changes

`Game` gains per-seat `tempFuriten [numSeats]bool`. The flag SHALL flip true when ALL hold: the seat's `concealed + just-discarded-tile` would form a yaku-bearing winning shape (`calc.Analyze != nil`), AND the seat did NOT submit a `ClaimRon` for that discard. The flag SHALL flip false when the seat draws on their next own turn (in `stepFromAwaitingDraw`). `IsFuriten(seat)` SHALL return true when EITHER the existing permanent-furiten condition holds OR `tempFuriten[seat]` is true.

`Bot` gains two helpers and a danger-aware discard path:

`Genbutsu(pond []tile.Tile, candidate tile.Tile) bool` — returns true when `candidate.ID` matches any tile ID in the seat's pond. (A discarded tile is permanently safe against that seat's hand.)

`SujiSafe(pond []tile.Tile, candidate tile.Tile) bool` — for non-honor candidates, returns true when the pond contains a 4 of the same suit AND `candidate.Rank()` is 1, OR contains a 6 of the same suit AND `candidate.Rank()` is 7. (A 4-discard says "no ryanmen on 1-2 or 5-6 needing 4"; symmetrically the 7 follows from a 6.) Honors and middle ranks 2/3/5/6/8 are never suji-safe.

`Bot.DangerAwarePickDiscard(hand []tile.Tile, danger map[uint8]int) int` — returns the discard index. `danger[id]` is a per-tile-ID penalty: 0 = safe (genbutsu), 1 = low (suji), 2 = unknown. The function blends the existing isolation score with `-danger[hand[i].ID]*K` where K is a large constant chosen so that any safe tile is preferred over any unsafe tile of equivalent isolation. When the `danger` map is empty (no opponent in riichi), the function falls back to the existing `PickDiscard` behavior.

The TUI's bot dispatcher (`dispatchBotDiscard` in `internal/play/play.go`) SHALL build the `danger` map by iterating opponent seats: for each seat with `riichiDeclared[seat] == true`, walk that seat's pond and mark genbutsu IDs with score 0, suji IDs with score 1; everything else stays at the default 2.

## Non-Goals

- Kabe (wall) reading: counting visible copies of a tile to infer "all 4 visible → can't be used in opponent's wait". Adds combinatorial complexity for marginal strength.
- Yomi (hand reconstruction): inferring opponent's likely waits from their discard tempo, suji choices, dora attraction. A whole research area; out of scope for hand-coded heuristics.
- Pre-riichi danger signals: a non-riichi bot with a fast, steep, all-numeric pond is dangerous too. v1 only reacts to formal riichi declarations.
- Push/fold judgment: bots always "push" — they pick the most-isolated-yet-safe tile rather than ever giving up tenpai progress to fold a tile to safety. Real play often requires breaking shape for safety.
- Multi-riichi conflict resolution: when two seats are both in riichi with conflicting suji-safety, v1 unions their dangers (max danger wins). No prioritization beyond that.
- Furiten across multiple turn cycles: temporary furiten lifts on the bot's next own draw; if they then pass on a third opponent's discard, this resets the temp-furiten flag (next pass re-arms it).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `game-loop`: extends `Furiten Query` with the temporary-furiten case (machi tile passed since last own draw). Adds the per-seat `tempFuriten` state to the engine. Extends `Bot Decision Strategy` with the danger-aware discard rule replacing the pure-isolation heuristic.

## Impact

- Affected specs: modified capability `game-loop` (Furiten Query + Bot Decision Strategy).
- Affected code:
  - Modified:
    - internal/game/turn.go (tempFuriten field, set on opponent-discard-of-machi-without-ron, cleared on own draw, IsFuriten union)
    - internal/game/bot.go (Genbutsu, SujiSafe, DangerAwarePickDiscard)
    - internal/play/play.go (dispatchBotDiscard builds danger map from riichi-declared opponents' ponds, calls DangerAwarePickDiscard)
    - internal/game/bot_test.go (tests for Genbutsu, SujiSafe, DangerAwarePickDiscard)
    - internal/game/furiten_test.go (tests for temporary furiten lifecycle: machi-passed → furiten, draw → cleared)
    - internal/play/play_test.go (test that bot avoids genbutsu-of-riichi-declarer when discarding)
  - New:
    - (none)
  - Removed:
    - (none)
