## Why

The rules engine is complete (`hand-calculator`) and the TUI skeleton renders a static layout (`play-screen`). The next milestone is **interactive play** — a real game state machine that deals 13 tiles to each of 4 players, runs a draw/discard turn cycle, supports calls (pon/chi/ron), drives dummy bot opponents, and ends a hand with a winning agari that the engine scores. This is the change that turns the project from "a calculator with a mockup" into "you can actually play a hand of riichi."

The interactivity has two motivations beyond the obvious one:

1. It exercises the engine in real situations (which will surface latent bugs in the calculator that the hardcoded test hands didn't catch).
2. It establishes the host code that future changes (smarter AI, trainer aids, networked multiplayer) all build on. Getting the state-machine boundary right now saves rework later.

## What Changes

- **New `internal/game` package** — a UI-free game state machine: 136-tile wall, dealing, dora indicator, four `Player` records (hand, called melds, discards, riichi state), turn cycle, and call resolution priority (ron > pon = open kan > chi).
- **Deterministic shuffling via `--seed N`** on `mahjong play`. No flag → cryptographically-random seed. Tests pin a seed for golden-game integration tests.
- **Bot strategy in `internal/game/bot.go`** — single tier for v1 ("Common calls + ron"):
  - **Discard**: pick the tile most isolated from neighbors in the same suit (terminals and honors are most isolated by definition; ties broken by lowest tile ID).
  - **Pon**: always call when an opponent discards a tile the bot has 2 of AND the tile is yakuhai (round wind, seat wind, or any dragon). For non-yakuhai pon, call with 50% probability when the bot is at shanten ≤ 2.
  - **Chi**: only from kamicha (bot's left-side opponent). When kamicha discards a tile that completes a 2-tile partial in the bot's hand, call chi with 40% probability.
  - **Ron**: call ron when the discarded tile completes a winning hand with at least one yaku. Uses existing `calc.Analyze`.
  - **Tsumo**: tsumo when drawn tile completes a winning hand with at least one yaku.
  - **Kan**: never. **Riichi**: never. (Both deferred — kan needs engine support, riichi needs hand-direction sense.)
- **Call-window UX in the TUI** — turn-locked, not real-time. After every opponent discard the loop pauses on a "claim window" footer prompt; only legal-call keys are live (`P` pon, `C` chi, `K` kan greyed always for v1, `R` ron, `Space` pass). No timeout.
- **Per-player discard zones** — replace the single centre pond with four zones, one per seat. Each zone wraps every 6 tiles into a new sub-row. Tiles render upright in all four zones (no rotation — terminals can't rotate). Centre region keeps round / honba / wall-count / dora indicator info.
- **ASCII compact tile form for ponds** — a 1-row-tall `[1m]` form used only in the four pond zones; the full 3-row boxed form remains for the player's hand. Without this the four ponds plus hand plus toimen tile-backs overflow the 80×24 budget in ASCII mode.
- **Engine wiring from TUI** — the play model holds a `*game.Game` pointer; on player tsumo / ron / call decisions it invokes `calc.Analyze` with full context (including Group C yaku state from the game loop).
- **Group C yaku detection** — extend `yaku.Context` with eight bool flags (`Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `DoubleRiichi`, `Tenhou`, `Chiihou`) and add corresponding detectors. The game loop populates these flags when calling `calc.Analyze` on a win. Three of the eight (Rinshan, Chankan, kan-related) cannot trigger in v1 because kan is unsupported, but their detectors exist and the flags exist on `Context` so the future kan-support change wires them in without an engine rewrite.
- **State-machine unit tests** — one test per legal turn transition (deal → first draw → discard → next-player draw, opponent-pon-interrupts, ron-on-discard, tenpai-at-exhaustive-draw, etc.).
- **Golden-game integration tests** — pin a seed, run a complete round (4 hands, east-only), capture the resulting event log (deals, discards, calls, wins) into a golden file, and compare. `go test -update` regenerates.
- **Manual smoke-test of the TUI** — same convention as the skeleton change.

## Non-Goals (optional)

- Smarter AI — danger awareness (folding when opponent declares riichi), tile-safety scoring, hand-direction reasoning. Deferred to a follow-up `add-smart-ai` change.
- Trainer aids — machi peek (`?` shows current waits and predicted yaku), furiten warning, illegal-call greying with engine-driven legality. Deferred to `add-trainer-aids`.
- Kan support — sankantsu / suukantsu yaku, kan additions to the fu table, the dead wall mechanics, rinshan and chankan triggers. Deferred to `add-kan-support`.
- Networked multiplayer — Tenhou-protocol or any other wire format. Out of scope.
- Game persistence and replay format — saving / loading partial games, exporting `.mjlog`. Out of scope.
- Bot-vs-bot self-play mode without a human seat. Out of scope.
- Real-time call timeouts — bots claim instantly, the human gets unbounded thinking time.

## Capabilities

### New Capabilities

- `game-loop`: A deterministic-when-seeded riichi game state machine — 136-tile wall, dealing, four-player turn cycle with call resolution priority (ron > pon = open kan > chi), per-seat hand and discard tracking, dora indicator state, and bot decision logic for opponents. Surfaces transitions as observable events that the TUI consumes for rendering and the test harness consumes for golden-game replay.

### Modified Capabilities

- `play-screen`: Replace the hardcoded-fixture rendering with live game state — per-player discard zones in place of the centre pond, real shanten/machi/yaku from engine queries, call-window prompts after opponent discards, an ASCII compact pond tile form alongside the existing full-boxed hand form.
- `hand-calculator`: Extend `yaku.Context` with the eight Group C state flags (`Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `DoubleRiichi`, `Tenhou`, `Chiihou`) and add eight detectors, six of which can trigger in this change (the two kan-dependent detectors exist but stay dormant until kan support lands).

## Impact

- Affected specs: new capability `game-loop`; modified capabilities `play-screen` and `hand-calculator`
- Affected code:
  - New:
    - internal/game/wall.go
    - internal/game/wall_test.go
    - internal/game/state.go
    - internal/game/state_test.go
    - internal/game/turn.go
    - internal/game/turn_test.go
    - internal/game/call.go
    - internal/game/call_test.go
    - internal/game/bot.go
    - internal/game/bot_test.go
    - internal/game/event.go
    - internal/game/golden_test.go
    - internal/play/pond.go
    - testdata/game/golden/.gitkeep
  - Modified:
    - cmd/play.go (add `--seed` flag, construct Game and pass to play.Model)
    - internal/play/play.go (replace fixture with live game state; add call-window handling; wire engine queries)
    - internal/play/render.go (add ASCII compact pond tile form)
    - internal/play/keys.go (call-window key bindings)
    - internal/riichi/yaku/yaku.go (extend Context, add eight Group C detectors)
    - internal/riichi/yaku/yaku_test.go (fixtures for the six in-scope Group C yaku)
    - internal/riichi/calc/calc.go (forward Group C flags from caller into yaku.Context)
