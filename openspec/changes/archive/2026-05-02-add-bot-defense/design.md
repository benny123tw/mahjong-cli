## Context

Smart bots already win, claim, and declare riichi. Their discard heuristic is `Bot.PickDiscard`, scoring each tile by isolation: `isolationScore` returns ~100 for fully-disconnected numerics, drops by adjacency, and floors at 1000 for honors. The bot picks the highest-scoring tile.

The danger gap: `dispatchBotDiscard` doesn't pass any board context to `PickDiscard`. It receives `m.game.Hand(seat)` and that's it. Adding danger awareness means flowing per-tile-ID risk scores from the dispatcher (which can read `riichiDeclared` and opponent ponds) into the discard scorer.

The furiten gap: `Game.IsFuriten(seat)` walks the seat's own pond against `hand.Machi`. It misses the temporary case — a winning tile passed by the bot since their last draw. The dispatcher already has the opportunity to detect this: when the bot evaluates ron and chooses NOT to claim, that decision moment is when temp-furiten arms.

Constraints:
- Determinism: bot decisions must remain replay-safe. The danger map is computed deterministically from observable state; suji/genbutsu are pure functions.
- Engine remains UI-agnostic. The `Genbutsu` and `SujiSafe` helpers live in `internal/game` since they're pure tile logic, but the danger-map ASSEMBLY (which needs to know which seats are riichi-declared) can live in either layer. Choice below.

## Goals / Non-Goals

**Goals:**

- Bots avoid discarding into a riichi declarer's likely waits, preferring genbutsu (100% safe) and suji (heuristic-safe).
- Temporary furiten is enforced: a bot that "saw" their winning tile pass cannot ron until next own draw.
- Danger awareness is opt-in via the dispatcher building a `danger` map; with an empty map the bot falls back to existing isolation behavior.
- Determinism preserved: same seed produces same game.

**Non-Goals:**

- Kabe (visible-tile counting), yomi (opponent hand reconstruction).
- Pre-riichi danger signals.
- Push/fold judgment beyond "prefer safer tiles when available".
- Multi-riichi prioritization (max-of-dangers union).
- Wide furiten generalization (only the tactical "passed on winning tile" case; not "drew a useless tile" or other edge cases).

## Decisions

### Genbutsu And Suji Live In `internal/game/bot.go`, Danger Map Assembly Lives In `internal/play/play.go`

`Genbutsu(pond, candidate)` and `SujiSafe(pond, candidate)` are pure tile-logic helpers — they take a pond slice and a tile and return a boolean. No game state, no seat awareness. They belong in the bot package as utility functions used by the dispatcher.

The danger map (`map[uint8]int` of tile-ID → risk score) requires reading `Game.riichiDeclared[]` and opponent `Game.Discards(seat)`. That's game-state coordination, which lives one level up in the play package's dispatcher. The dispatcher iterates `[]Seat{East,South,West,North}`, picks the riichi-declared opponents, builds the danger map, then calls `bot.DangerAwarePickDiscard(hand, danger)`.

**Alternative considered:** Put the danger-map assembly in `Game.DangerMap(forSeat) map[uint8]int`. Rejected — game/turn.go already has many responsibilities; adding a discard-strategy helper to it pushes UI concerns (which seats are bots) into the engine. Keep the engine focused on rules; keep strategy in the play layer.

### Danger Penalty Constant K Is Chosen To Dominate Isolation Differences

Isolation scores range from ~85 (heavily-connected middle tile) to 1000 (honor with no copies). Danger levels are 0/1/2.

To make ANY safe tile beat ANY unsafe tile of equivalent shape, the penalty per danger level must exceed the maximum isolation gap. With honors at floor 1000 the gap can be ~915 between an honor and a fully-connected numeric. So `K = 2000`:
- danger 0 (genbutsu): score = isolation - 0 = unchanged
- danger 1 (suji): score = isolation - 2000 (still better than danger 2 but loses big to genbutsu)
- danger 2 (unknown): score = isolation - 4000

This guarantees ordering: genbutsu > suji > unknown, regardless of isolation. Among same-danger tiles, isolation wins (existing behavior).

**Alternative considered:** Multiplicative penalty (score *= 0.5 per danger level). Rejected — multiplicative interacts poorly with the honor-floor (honors at 1000 → still 250 at danger 2, beats numeric at 100 unsafe). Additive with a dominating constant is simpler and predictable.

### Suji Coverage Is Limited To 1↔4 And 6↔7 Pairs

Standard riichi suji theory:
- A 4 in pond → 1 and 7 are suji (no 23-needs-4 ryanmen, no 56-needs-4 ryanmen).
- A 5 in pond → 2 and 8 are suji (similarly).
- A 6 in pond → 3 and 9 are suji.

We implement only the 4→1/7 and 6→1/7 forms... actually wait, that's 4→1 (covers 23-need-4, but 1 is also covered by 4→7 since 56-need-4 doesn't reach 1). Let me reconsider.

Riichi suji rules:
- 1 is suji-safe if pond has 4 (covers 2-3-needs-4 ryanmen).
- 7 is suji-safe if pond has 4 (covers 5-6-needs-4 ryanmen).
- 2 is suji-safe if pond has 5 (covers 3-4-needs-5).
- 8 is suji-safe if pond has 5.
- 3 is suji-safe if pond has 6.
- 9 is suji-safe if pond has 6.

So the full table is `(safe-rank, requires-pond-rank)`: (1,4), (7,4), (2,5), (8,5), (3,6), (9,6). Implement all six pairs in `SujiSafe`.

**Alternative considered:** Implement only (1,4) and (7,4) per the proposal's literal "4 or 6" wording. Rejected — the proposal under-specified; the full set is the same code-shape (a lookup table) and noticeably stronger.

The proposal text will be amended in the spec delta to reflect the full table.

### Temporary Furiten Arms In The Engine, Not The Dispatcher

The natural site to detect "bot saw winning tile, chose not to ron" is `dispatchBotClaims`: when the bot's ron evaluation returns "could ron" but the dispatcher picks pass anyway (currently never — but if a future heuristic adds "fake-pass" or yakuless-pass, it lands here).

In v1 today, bots ALWAYS ron when they can. So temp-furiten only arms in one specific case: yaku-less win on opponent's discard. Today's `dispatchBotClaims` evaluates `calc.Analyze` — if non-nil, ron. So yakuless-on-bot's-side never gets a "pass" decision; the calc returns nil and the bot doesn't even consider ron.

But the FURITEN rule still applies: even on a yakuless winning shape, the seat is locked into temp-furiten. The standard riichi rule: the tile is the seat's machi; whether it has yaku isn't relevant to furiten. (Real-world: a player passes on a yakuless win to avoid the chombo penalty for declaring without yaku — but they're now furiten.)

So: dispatch logic must arm temp-furiten when `concealed + discard` would be a winning shape (`hand.IsWinning`) regardless of yaku. The check is `hand.IsWinning(hand.Hand{Concealed: concealed, Winning: discard})` rather than `calc.Analyze != nil`.

Implementation: in `Game.stepFromAwaitingClaims`, before resolving the claim, walk all non-discarder seats and arm `tempFuriten[seat]` if `hand.IsWinning(...)` would have been true AND the seat did NOT submit a ron claim. Then resolve normally.

**Alternative considered:** Arm temp-furiten in `dispatchBotClaims` (TUI side). Rejected — the engine should enforce the furiten rule for all seats, not depend on the TUI dispatcher. Moving the check to `stepFromAwaitingClaims` makes it apply uniformly to humans (who pass via Space) AND bots.

### Tests Cover Helpers, Engine Lifecycle, And Integrated Dispatcher Behavior

- `internal/game/bot_test.go` adds `TestBotGenbutsu` (pond contains/doesn't contain candidate), `TestBotSujiSafe` (full 6-pair table), `TestBotDangerAwarePickDiscard` (genbutsu beats suji beats unknown beats nothing).
- `internal/game/furiten_test.go` adds `TestTempFuritenArmsOnPassedWin` (plant a tenpai bot, opponent discards a winning tile, bot passes, IsFuriten now true) and `TestTempFuritenClearsOnNextOwnDraw` (after the above, bot draws → IsFuriten reverts to permanent-only check).
- `internal/play/play_test.go` adds `TestBotDispatchAvoidsGenbutsu` (bot at SeatNorth, North's hand can discard either a tile in East's pond or a tile NOT in East's pond; East is in riichi; bot should pick the genbutsu).

## Risks / Trade-offs

[Risk: blending danger and isolation with a hardcoded K=2000 produces brittle ordering — if isolation scores ever exceed K, danger no longer dominates] → Mitigation: K is chosen to exceed the actual isolation range (max ~1000 for honors). Add a unit test asserting the ordering invariant: any safe tile beats any unsafe tile of any isolation score.

[Risk: temp-furiten arming on `IsWinning` (not `calc.Analyze`) means yakuless winning passes will arm furiten — but is that correct riichi rules?] → Mitigation: yes, in real riichi the seat is furiten regardless of whether the win has yaku. The "I can't claim because no yaku" loss is permanent for that round. Confirmed by referencing standard riichi furiten tables.

[Risk: temp-furiten interacts poorly with multiple opponent discards before own draw — does the flag re-arm on each pass?] → Mitigation: yes, the engine arms temp-furiten on EVERY pass over a winning tile. Subsequent same-tile-id passes are idempotent (already armed). Different machi tiles passed in sequence all contribute to "stay armed". Cleared only on own draw.

[Risk: bots don't fold tenpai for safety. A bot at tenpai with a gun-pointed-at-you tile in hand will discard the tile if it's still the best-scored in their hand. Some scenarios this misses real-world strategy] → Mitigation: documented Non-Goal. Push/fold judgment ships in a future change if needed.
