## Context

Two prior changes have shipped: `add-hand-calculator` (rules engine + `mahjong calc`) and `add-tui-skeleton` (bubbletea v2 layout with hardcoded fixture). The skeleton was deliberately scoped to layout-only — model, render, keymap, no game state. This change is the bridge between the two: it introduces real game state, four-player turn flow, opponent simulation, and call resolution, then wires the existing rules engine into the existing TUI shell.

The author is a Taiwanese-mahjong player learning Japanese (riichi). The bot strategy and trainer-aid scope are calibrated for that goal — bots should make calls so the player practices the JP-vs-TW rule differences (especially chi-from-left-only), but bots should not yet be skilled enough to reliably win against a learner.

## Goals / Non-Goals

**Goals:**

- Ship a complete riichi hand from deal to win that can be played at `mahjong play` against three dummy bots.
- Establish the game-state-machine boundary so future changes (smart AI, trainer aids, networked multiplayer, kan support) extend it without rewriting.
- Cover the six Group C yaku that can trigger without kan support, with detectors that match the existing yaku-detector pattern.
- Provide deterministic playback via `--seed` so games can be reproduced for testing and bug reports.

**Non-Goals:**

- Smart AI (danger awareness, hand-direction reasoning) — `add-smart-ai`.
- Trainer aids (machi peek, furiten warning, illegal-call greying) — `add-trainer-aids`.
- Kan support and the kan-dependent yaku — `add-kan-support`.
- Networked multiplayer, replay export / import — out of scope.
- Bot-vs-bot self-play, headless practice mode — out of scope.
- Real-time call timeouts — turn-based throughout.

## Decisions

### Game state lives in `internal/game/`, with zero TUI dependencies

Mirrors the engine-vs-UI rule established in `add-hand-calculator`: `internal/game` may import `internal/riichi/{tile,hand,yaku,calc}` but MUST NOT import `internal/play`, `cmd/`, `bubbletea`, or `lipgloss`. The TUI holds a `*game.Game` pointer and observes state through query methods; mutations go through state-transition methods that return events. This keeps the game loop testable without spinning up a TUI and lets golden-game tests run as plain `go test`.

Alternatives considered:
- **Game state inside the bubbletea Model**: rejected; tangles game logic with rendering, makes integration tests harder to write, blocks future server-side game runner.
- **Public `pkg/game/`**: rejected; same reasoning as `internal/riichi/` — keep API private until stable.

### Turn cycle as an explicit state machine with named states

Rather than implicit step-by-step code, the game is a machine over five states:

- `StateAwaitingDraw{Player}` — turn just rotated, expect a draw.
- `StateAwaitingDiscard{Player}` — drew a tile, expect a discard or tsumo or riichi.
- `StateAwaitingClaims{Discard, Discarder}` — someone discarded; other players have a window to claim.
- `StateRoundOver{Outcome}` — agari (win) or ryuukyoku (exhaustive draw).
- `StateGameOver{Standings}` — round terminated past hanchan boundary (v1: just one round, single state instance).

Transitions are pure functions `(state, event) → (state, []event)`. This shape makes the state machine trivially testable (one test per legal transition) and unambiguous about what is and isn't allowed mid-turn.

Alternatives considered:
- **Implicit cycle in a goroutine**: rejected; harder to test, harder to drive from TUI updates which are message-based.
- **Single big switch in `Update`**: rejected; tangles game rules with bubbletea concerns.

### Call resolution priority and the claims window

After every discard, the state machine enters `StateAwaitingClaims` with a fixed turn-order priority for resolving overlapping claims:

1. **Ron** by any player who can declare it. Ties resolved by **head-bump rule** — the player closest to the discarder going right (shimocha first) wins. (This is a v1 simplification; full riichi has variations on multi-ron rules.)
2. **Pon / open kan** by any player. Higher priority than chi.
3. **Chi** by kamicha (the player whose turn would naturally come next).

Bot players resolve their claim instantly; the human player gets a synchronous prompt in the TUI footer. The state machine collects all claims in priority order and picks the winner.

Alternatives considered:
- **Strict double-ron not allowed (atamahane)**: chosen for v1 — head-bump is closer to most casual riichi rules and avoids the multi-ron payment edge cases.
- **Claims via async channels**: rejected; bubbletea is single-threaded, async would force goroutines and timing complexity for marginal gain.

### Bot strategy: single-tier "Common calls + ron"

Bots in v1 play one strategy, hand-coded:

- **Discard**: pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere in hand). Honors and terminals score highest. Tiebreak: lowest tile ID.
- **Pon**: yakuhai pon always (any tile that's seat wind, round wind, or any dragon — when the bot has 2). Non-yakuhai pon at 50% probability when shanten ≤ 2.
- **Chi**: from kamicha only, 40% probability when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in the bot's hand.
- **Kan**: never.
- **Riichi**: never.
- **Ron**: always when the discarded tile completes a yaku-bearing winning hand.
- **Tsumo**: always when the drawn tile completes a yaku-bearing winning hand.

The probabilities use a deterministic PRNG seeded from the same seed as the wall, so games reproduce exactly.

Alternatives considered:
- **Truly dumb bots (no calls)**: rejected; the player would never see calls and would miss practicing the JP-specific call rules (especially chi-from-left).
- **Full-tier bots** (kan, riichi, defense): rejected as too complex for v1; smart-AI deserves its own change.

### Deterministic shuffle via `--seed N`

`mahjong play [--seed N]`. Without the flag the seed is derived from the OS PRNG and printed at game start so the player can reproduce a hand if something interesting happens. With the flag, the shuffle plus all bot probabilistic decisions use that seed — same seed yields byte-identical game sequence.

The shuffle uses Go's `math/rand/v2` (PCG) seeded from the user's seed for the wall and for bot decisions. Crypto-random would also work but PCG is cheap and deterministic-friendly.

Alternatives considered:
- **`crypto/rand` with no seeding**: rejected; precludes deterministic tests and bug reports.
- **OS PRNG without printing the seed**: rejected; loses reproducibility when the player hits something interesting.

### Per-player pond layout in 80×24

Each player's discards live in a region in front of their seat. The four regions wrap into a sub-row every 6 tiles (riichi convention). Tiles render upright in all four regions because terminals can't rotate text — kamicha and shimocha discard rows are not visually rotated, just placed on the sides.

Layout sketch (Unicode mode):

```
Row  0: status (round / honba / wall / dora / scores)
Row  1: (spacer)
Rows 2-3: toimen tile-backs row + label
Rows 4-7: toimen discards (up to 4 sub-rows of 6 = 24 tiles before overflow)
Row  8: (spacer)
Rows 9-13: kamicha discards (left col) │ centre info (round, dora, wall) │ shimocha discards (right col)
Row  14: (spacer)
Rows 15-18: your discards (4 sub-rows of 6)
Row  19: (spacer)
Row  20: your hand (1 row Unicode / 3 rows ASCII)
Row  21-23: footer with active call-window prompt or turn label
```

ASCII mode uses the **compact pond tile form** — `[1m]` style, 1 row tall, 4 columns wide — only for the four discard zones. The full 3-row boxed form is reserved for the player's hand. Without the compact form, the four pond zones plus toimen tile-backs plus the player's hand exceed 24 rows in ASCII mode.

Centre region (the open space inside the four ponds) shows round wind (E/S), honba count, wall-remaining count, and the active dora indicator tile. No pond contents in the centre — discards moved out to per-player zones.

Alternatives considered:
- **Vertical-rotated tiles for kamicha/shimocha**: rejected; faking rotation in monospace is brittle and ugly.
- **Drop ASCII mode entirely**: rejected; ASCII is the reliable cross-terminal escape hatch when Unicode rendering misbehaves.
- **Reflow on terminal resize**: rejected; deferred per the skeleton's stored-but-ignored window-size policy.

### Group C yaku integration via state flags on `yaku.Context`

`yaku.Context` gains eight bool fields: `Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `DoubleRiichi`, `Tenhou`, `Chiihou`. Each flag is populated by the game loop when calling `calc.Analyze` on a winning hand. Detectors are added to the existing `Detectors()` slice in `internal/riichi/yaku/yaku.go` and each is a one-liner reading its flag.

The kan-dependent flags (`Rinshan`, `Chankan`) and their detectors ship in this change but never get set to true in v1 because no game state path produces them. This means the future kan-support change wires them in without an engine rewrite — a cost-free forward-compatibility win.

Alternatives considered:
- **Separate `yaku.GameContext`** struct: rejected; bloats the API for marginal organization gain.
- **Defer Group C entirely until kan**: rejected; six of the eight yaku trigger in v1 game-loop and shipping detectors-only-and-lights-off would mean an incomplete winning evaluation in the interim.

### Bot turn timing: tea.Tick-paced, not synchronous

When it's a bot's turn, the model returns a `tea.Tick` command for ~250ms before the bot acts. This gives the player time to see the previous discard before the next one happens, without making the game feel laggy. Multiple consecutive bot turns chain via tea.Cmd, so a full bot round trip (discard → next-bot-draw → next-bot-discard) takes about a second.

Alternatives considered:
- **Synchronous bot turns inside Update**: rejected; user can't see intermediate states, feels like a state snap.
- **Real-time animation per draw/discard**: rejected; over-engineered for v1.

### Test strategy: state-machine unit + golden-game integration + manual smoke

Three layers, mapped to where each kind of bug surfaces:

1. **State-machine unit tests** — one test per legal transition. Fast, deterministic, no UI. Covers "from `StateAwaitingDiscard` after discard, with no claims, transitions to `StateAwaitingDraw` for next player." Lives in `internal/game/state_test.go` and friends.
2. **Golden-game integration tests** — one full round (4 east hands) at a fixed seed. Capture a textual event log (`deal seed=N → hand 1: P0 discards 4m, ..., P2 ron on 5p, P0 pays 5200`) into `testdata/game/golden/`. `go test -update` regenerates after intentional changes. Lives in `internal/game/golden_test.go`.
3. **Manual TUI smoke-test** — same convention as `add-tui-skeleton`. Author plays `mahjong play --seed 42` and confirms the layout renders, calls work, the engine scores wins correctly. No automated TUI tests in this change (deferred to a focused `add-tui-tests` change if it ever proves needed).

The golden-game tests are the load-bearing safety net: they catch any regression in dealing, turn flow, claim resolution, bot decisions, or yaku detection by replaying a seeded game and diffing the event log.

## Risks / Trade-offs

- **Multi-ron edge cases.** With head-bump rule we punt on the riichi multi-ron payment table. → Accepted; full multi-ron is its own complexity; documented as a v1 simplification.
- **Bot strategy is rigid.** Probabilities are hand-tuned constants. The bots will feel predictable after a few games. → Accepted for v1; smart-AI is a follow-up change with hand-direction logic.
- **Layout overflow at 80×24 in ASCII mode** is genuinely tight. The compact pond form gives ~6 rows per pond × 4 ponds = 24 rows of pond alone, plus 3 rows of hand = 27 rows. The four ponds will need to share rows or limit visible discards. → Mitigation: cap visible discards at 12 per pond (2 sub-rows of 6); older discards scroll off the top with a "+N earlier" indicator.
- **tea.Tick pacing might feel slow on bot calls.** A bot pon interrupting your turn at 250ms × N bot decisions could chain to ~1s of waiting. → Accepted; if the user complains, easy follow-up to tune.
- **Group C yaku correctness.** `Tenhou` (dealer wins on initial dealt hand) is exotic enough that it might never trigger in actual play and a bug could go unnoticed. → Mitigation: explicit unit test that constructs a tenhou hand fixture and asserts the detector triggers.
- **`--seed` makes runs reproducible but every game is the same if seed is constant.** → Accepted; that's the point for tests; for play, we default to OS-random and print the seed for after-the-fact reproduction.
- **State machine + bot strategy + TUI integration + Group C yaku is a lot of surface for one change.** Tasks list will be near the upper bound. → Mitigation: structure tasks tightly per-decision; if it overflows ~18 tasks during proposal review we'll split off Group C yaku into a follow-up. The current target is to keep it at 16-18 tasks.
