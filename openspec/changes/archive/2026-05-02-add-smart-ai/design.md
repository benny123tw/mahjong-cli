## Context

`internal/play/play.go`'s `handleBotTick` currently dispatches three states with minimal logic:

```
AwaitingDraw → InputDraw{}
AwaitingDiscard → InputDiscard{Index: bot.PickDiscard(hand)}
AwaitingClaims → InputResolveClaims{Claims: nil}  // unconditional pass
```

`Bot` already has `ShouldPon`, `ShouldChi`, `ShouldKan` (always false), `ShouldRiichi` (always false), `PickDiscard`. The engine's `stepFromAwaitingClaims` handles `ClaimRon`, `ClaimPon`, `ClaimChi` correctly; `ResolveClaims` enforces ron > pon > chi priority and the head-bump tiebreak.

`Game.IsFuriten(seat)` returns true iff any machi tile is in the seat's own pond. The current `ClaimRon` branch gates furiten only when `winner == HumanSeat` — bots could theoretically ron through furiten today, but they never ron at all because nothing dispatches `ClaimRon` for them.

`add-human-agari` just plumbed `riichiDeclared`, `ippatsuLive`, `doubleRiichi`, and `scores` per seat. The riichi state machine works for any seat (the engine doesn't gate on seat identity), so bots inherit the full riichi infrastructure for free — they just need to opt in by submitting `InputDiscard{Riichi: true}`.

Constraints:
- Determinism: same seed produces byte-identical event log. Bot decisions use `Wall.Rand()` (seeded PRNG); calls to `bot.ShouldPon`/`ShouldChi` already consume from this stream. Adding bot ron / tsumo / riichi must NOT add new RNG consumption sites for non-bot-decision events.
- Engine remains UI-agnostic (no imports from internal/play in internal/game).

## Goals / Non-Goals

**Goals:**

- Bots win when they have a winning hand: tsumo on draw, ron on discard.
- Bots call pon/chi when their existing heuristics return true.
- Bots declare riichi when tenpai-after-discard, concealed, with funds and wall.
- Furiten gate applies to every ron claim, not just the human's.
- Determinism is preserved: a fixed seed produces a fixed game.

**Non-Goals:**

- Bot defense (folding safe tiles when an opponent is in riichi).
- Bot tile-choice strategy beyond "first tenpai-leaving index" for riichi.
- Pao / sekinin-barai.
- Bot temporary furiten (passing on a winning discard makes them furiten until next draw).
- Bot kan.
- Bot ippatsu / double-riichi awareness in heuristics (engine still detects them correctly when the bot wins).

## Decisions

### Bot Tsumo Check Runs Before The Discard Decision

In `handleBotTick`'s `AwaitingDiscard` branch, the order is: (1) check tsumo via `calc.Analyze` on the bot's 14-tile hand → submit `InputDeclareTsumo` if non-nil; (2) else check riichi via the new `ShouldRiichi` → submit `InputDiscard{Index: chosenIdx, Riichi: true}` if true; (3) else fall back to existing isolation-heuristic discard.

Tsumo before riichi is correct: a winning hand is always preferred over declaring riichi (and waiting for next draw). The engine's `InputDeclareTsumo` validates the win via `calc.Analyze` and returns `ErrYakulessWin` if no yaku — at which point the bot falls through to the riichi/discard path. The yakuless-win rejection means the engine's contract is the source of truth; the bot doesn't need to re-check yaku.

**Alternative considered:** Add `ShouldTsumo` and `ShouldRon` helpers on `Bot` mirroring `ShouldPon`. Rejected — these helpers would just call `calc.Analyze` directly with no additional logic. Inlining the calc call in `handleBotTick` keeps the bot package small and avoids a thin pass-through method.

### Bot Ron Sits In Claims Branch Per-Seat, Submitted Through The Existing Resolver

`AwaitingClaims` iterates seats East/South/West/North (skipping the discarder). For each non-discarder bot, the dispatcher computes a `Claim`: ron first (calc + furiten check), then pon (`ShouldPon`), then chi (`ShouldChi` from kamicha only), then pass. The complete map is submitted to `InputResolveClaims` in one call; `ResolveClaims` enforces ron > pon > chi priority and the head-bump tiebreak.

This means the dispatcher does NOT short-circuit on the first ron — it collects all claims and lets the resolver pick the winner. Necessary because two bots could both ron the same tile (head-bump applies) or one could pon while another rons (ron wins).

**Alternative considered:** Short-circuit on the first detected ron. Rejected — the resolver is the canonical priority arbiter and exists for exactly this situation. Pre-empting it would duplicate priority logic and break head-bump.

### Bot Riichi Heuristic Picks The First Tenpai-Leaving Index

`ShouldRiichi(hand, scores, wallRemaining)` returns `(declare bool, tileIdx int)`. Implementation:

1. Validate preconditions: `len(hand) == 14`, score ≥ 1000, wallRemaining ≥ 4. Return `(false, 0)` on any miss. (The engine also re-validates these — this is just to skip the inner loop when obviously illegal.)
2. For `idx := 0; idx < len(hand); idx++`: build `postDiscard` by removing `hand[idx]`, run `hand.Shanten(postDiscard)`. If 0, return `(true, idx)`.
3. Return `(false, 0)` if no index produces tenpai.

The bot caller checks `bot.IsHandOpen` (via the engine's `IsHandOpen(seat)`) before invoking — `ShouldRiichi` itself takes only the hand slice and doesn't know about open melds.

**Alternative considered:** Score each tenpai-leaving discard by wait quality (count of remaining live tiles for each machi tile) and pick the highest. Rejected — meaningfully harder to implement (requires tile-counting from full state), and adds complexity that yields a 5-10% strength improvement at best. Out of scope; ships in a future bot-strategy refinement.

**Alternative considered:** Random tile choice from the set of tenpai-leaving indices. Rejected — the seeded PRNG is already busy; introducing an extra random call here would shift the deterministic event stream and require golden re-regeneration. Deterministic "first index" preserves byte-identical replay.

### Furiten Gate Applies To All Seats — Drop The Human-Only Guard

In `stepFromAwaitingClaims`'s `ClaimRon` branch, change `if winner == HumanSeat && g.IsFuriten(winner)` to `if g.IsFuriten(winner)`. Bots are now subject to the same permanent-furiten gate as the human.

Note: `IsFuriten` requires `len(hand) == 13` for the seat. At the moment of ron, the seat has not yet appended the discard tile, so the hand is exactly 13 — the check is well-defined.

**Alternative considered:** Add a separate gate variable like `enforceFuritenForBots bool`. Rejected — there's no scenario where bots should be allowed to ron through furiten. The rule is universal in real riichi.

### Bot Pon/Chi Probabilistic Decisions Use The Existing PRNG; No New Consumption Sites

`Bot.ShouldPon` and `Bot.ShouldChi` already consume `Bot.Rng` for their probability rolls. `Bot.ShouldRiichi`'s new implementation is fully deterministic (no random tile choice), so it doesn't add to the stream. Bot tsumo and bot ron are deterministic checks on `calc.Analyze` — no random calls.

The result: a fixed seed run produces a fixed event log. The existing `golden_test.go` does NOT exercise calls (it discards index 0 every turn, no claims), so smart-ai changes do not regenerate the seed-42 golden fixture.

**Alternative considered:** Add a "claim chance" probability to bot ron (e.g., bot rons 95% of the time, fakes 5% to model uncertainty). Rejected — bots in v1 are deterministic up to existing RNG sites; adding randomness is out of scope.

### Tests Cover Both Engine-Level Bot Helpers And Play-Level Dispatch

Engine tests in `internal/game/bot_test.go` cover `ShouldRiichi` (legal, illegal-because-funds, illegal-because-wall, illegal-because-not-tenpai, illegal-because-open). Play tests in `internal/play/play_test.go` cover `handleBotTick` for: bot tsumo on a planted winning hand, bot ron on an opponent's discard, bot riichi when tenpai, furiten-blocks-bot-ron.

The engine-level tests stay isolated from the TUI. The play-level tests drive the model through `Update(BotTickMsg{})` and assert resulting state.

## Risks / Trade-offs

[Risk: bot ron evaluation is `O(seats × calc.Analyze)` per claim window. `calc.Analyze` is non-trivial — it enumerates partitions and scores yaku. For a 4-seat game this is 3 evaluations per discard, 3 × ~70 discards = ~210 per round] → Mitigation: `calc.Analyze` is fast (microseconds in benchmarks). At 250ms bot-tick cadence the time is invisible to the player. If profiling reveals hot-spotting, cache `IsWinning` checks (which short-circuit before full scoring).

[Risk: bot riichi tile-choice "first tenpai" produces poor riichi declarations — the bot might lock in on a weak wait when a strong wait was available with a different discard] → Mitigation: documented as v1 limitation. Players grumbling about "the bot took a 1-out wait when a 6-out was available" gets a follow-up bot-strategy change.

[Risk: with bots ronning, the "human plays defensively forever" exploit is gone — but no one has defended against bot ron in this codebase. Test setup involving deal-in-by-human now has to account for the possibility of a bot ronning] → Mitigation: existing tests that need "human discards safely without ron" can plant the human's hand to ensure no bot is at a winning wait, or use `SetTestHand` for opponent seats to force their hands non-tenpai.

[Risk: bot temporary furiten is NOT implemented; a bot that passed on a winning discard one turn ago and rons on a different opponent's discard now is technically illegal in real riichi. v1 ignores this] → Mitigation: documented Non-Goal. Temporary furiten lands in `add-bot-defense` along with hand-reading.
