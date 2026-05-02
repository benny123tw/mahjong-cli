## 1. Wall Construction and Dealing

- [x] 1.1 Write `internal/game/wall_test.go` covering Wall Construction and Dealing — wall length 136, exact tile inventory (4 copies each of the 34 tile types; per spec, v1 ships no red fives — those land in `add-akadora-toggle`), seeded shuffle determinism (same seed → same order), and dealing 13 to each of 4 players + 14th to dealer. Tests SHALL fail before implementation (TDD red).
- [x] 1.2 Implement `internal/game/wall.go` to satisfy Wall Construction and Dealing — `NewWall(seed int64) *Wall`, `Deal()` returning four 13-tile hands plus the dealer's 14th draw, dora-indicator pointer, and live-wall remaining count. Implements the deterministic shuffle via `--seed N` design decision (the same `*rand.Rand` is exposed for the bot in task 4.2). Make all tests from 1.1 pass (TDD green).

## 2. Turn Cycle State Machine

- [x] 2.1 Write `internal/game/state_test.go` exercising the Turn Cycle State Machine — table-driven tests for each transition realizing the design decision "Turn cycle as an explicit state machine with named states" (`AwaitingDraw → AwaitingDiscard` on draw, `AwaitingDiscard → AwaitingClaims` on discard, `AwaitingClaims → AwaitingDiscard` when pon/chi resolves, `AwaitingClaims → AwaitingDraw` (next player) when no claim, `AwaitingDraw → RoundOver` when the live wall is exhausted, `AwaitingClaims → RoundOver` when ron resolves). Tests SHALL fail before implementation.
- [x] 2.2 Implement `internal/game/state.go` and `internal/game/turn.go` for the Turn Cycle State Machine — defining the `Game` struct, the five `GameState` variants from design, and the `Step(input Input) (Event, error)` advancement function. This realizes the architectural decision that the game state lives in `internal/game/`, with zero TUI dependencies (the package SHALL NOT import bubbletea, lipgloss, or anything under `internal/play`). Make all tests from 2.1 pass.

## 3. Call Resolution Priority

- [x] 3.1 Write `internal/game/call_test.go` covering Call Resolution Priority and the design decision "Call resolution priority and the claims window" — ron beats pon and chi; pon and open kan tie at the same priority and chi loses to both; with multiple ron claimants the head-bump rule SHALL select the closest seat counter-clockwise from the discarder; passing all claims advances to the next player.
- [x] 3.2 Implement `internal/game/call.go` to satisfy Call Resolution Priority — `ResolveClaims(claims []Claim) Resolution` plus helpers for legal-call detection (a player can pon if they have ≥2 of the discarded tile; chi only from kamicha and only when the discard plus two of the player's tiles make a sequence; ron requires a winning shape with at least one yaku). Make all tests from 3.1 pass.

## 4. Bot Decision Strategy

- [x] 4.1 Write `internal/game/bot_test.go` covering Bot Decision Strategy and the design decision "Bot strategy: single-tier "Common calls + ron"" — discard heuristic (most-isolated tile, ties broken by lowest tile ID), pon always-on for yakuhai triplets, pon 50% probability for non-yakuhai triplets when shanten ≤ 2 (use a fixed RNG seed to make probabilistic branches deterministic), chi 40% probability from kamicha only, ron always when winning with at least one yaku, never kan, never riichi.
- [x] 4.2 Implement `internal/game/bot.go` to satisfy Bot Decision Strategy — `Bot.Decide(ctx GameContext) Action` that consumes the same `*rand.Rand` as the wall (so `--seed` covers both shuffle and bot decisions). Make all tests from 4.1 pass.

## 5. Round Termination and Outcome

- [x] 5.1 Write tests in `internal/game/state_test.go` (extend) covering Round Termination and Outcome — exhaustive draw enumerates tenpai players for noten penalty, ron transitions cleanly to RoundOver with the winner and the discarder recorded, tsumo transitions cleanly to RoundOver with the winner recorded.
- [x] 5.2 Extend `internal/game/state.go` and add `internal/game/event.go` with the event log (Deal, Draw, Discard, Call, Win, RoundEnd) and the `Outcome` struct, satisfying Round Termination and Outcome. Make all tests from 5.1 pass.

## 6. Group C Game Context Flags and Yaku Detection

- [x] 6.1 Write fixtures in `internal/riichi/yaku/yaku_test.go` exercising the new entries in Yaku Detection — V1 Set for the Group C yaku, and exercising Group C Game Context Flags — Ippatsu (with and without a flag, and disallowed when open), Double riichi (suppresses regular riichi), Haitei (tsumo only), Houtei (ron only), Tenhou (dealer-only), Chiihou (non-dealer with no calls), and the dormant Rinshan / Chankan detectors that match only when the flag is forced on. Each fixture SHALL fail until the detector is implemented.
- [x] 6.2 Extend `yaku.Context` in `internal/riichi/yaku/yaku.go` to satisfy Group C Game Context Flags and the modified Yaku Detection — V1 Set, realizing the design decision "Group C yaku integration via state flags on `yaku.Context`". Add eight bool flags (`Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `DoubleRiichi`, `Tenhou`, `Chiihou`) and eight Group C detectors that consult the flags plus existing concealment / win-type checks. Add the rule that `DoubleRiichi` suppresses `Riichi`. Make fixtures from 6.1 pass; the existing 18-yaku suite SHALL stay green.
- [x] 6.3 Update `internal/riichi/calc/calc.go` to forward Group C flags from caller-supplied context into `yaku.Context` so the existing `mahjong calc` CLI continues to work with new flags exposed via flag-only override (no new CLI surface in this change — flags default to `false` in `cmd/calc.go`).

## 7. Play Screen Layout and Tile Rendering Strategy

- [x] 7.1 Write `internal/play/render_test.go` covering Tile Rendering Strategy — the existing Unicode and ASCII boxed forms remain unchanged, the new ASCII compact form produces exactly 4 columns × 1 row in `[1m]` style, and the new compact renderer SHALL be selected only when `--ascii` is active.
- [x] 7.2 Add an `ASCIIPondRenderer` to `internal/play/render.go` producing the 4×1 compact form. Add `internal/play/pond.go` rendering a per-seat pond zone of up to 12 most-recent discards in 6-wide sub-rows with a `+N earlier` overflow indicator. Replace the centre pond fixture in `internal/play/play.go` to satisfy Play Screen Layout — four per-seat zones positioned per the design decision "Per-player pond layout in 80×24" (toimen above, your zone below, kamicha left, shimocha right). Make tests from 7.1 pass.

## 8. Engine Wiring, Call Window, and Keybinding Map

- [x] 8.1 Write integration tests in `internal/play/play_test.go` covering Engine Wiring For Game State, Call Window Prompt, and the active-action portion of Keybinding Map — model accepts a `*game.Game` pointer and reflects its hand and discards, pressing `?` calls `hand.Shanten`/`hand.Machi` and caches the result until the hand mutates, pressing `T` on a winning drawn tile invokes `calc.Analyze` with full Group C context, pressing `R` on a yakuless winning shape gets rejected without state advance, the call-window footer renders only legal-call keys and `Space` always passes.
- [x] 8.2 Replace the fixture machinery in `internal/play/play.go` to retire Hardcoded Fixture For Display and satisfy Engine Wiring For Game State, Keybinding Map, and Call Window Prompt — drop `fixtureHand`, replace the `hand` field with `game *game.Game`, route `tea.KeyMsg` through `Game.Step` for state transitions, gate footer rendering on `game.State()`, and wire `?` / `T` / `R` / `P` / `C` / `Space` to engine calls. Make tests from 8.1 pass.

## 9. Bot Turn Pacing and Play Subcommand Launch

- [x] 9.1 Add a `tea.Tick`-paced bot-action message in `internal/play/play.go`, realizing the design decision "Bot turn timing: tea.Tick-paced, not synchronous" — when game state is `AwaitingDraw` or `AwaitingClaims` for a bot seat, a 250 ms tick delivers the bot's decision back into the model, advancing state via `Game.Step` and emitting a fresh `tea.Tick` if more bot actions follow. Tests in `play_test.go` SHALL exercise this with a mocked tick to assert the tick is scheduled and bot actions advance state without user input.
- [x] 9.2 Update `cmd/play.go` to satisfy Play Subcommand Launch and realize "Deterministic shuffle via `--seed N`" — add the `--seed N` flag (default 0 = OS-random; non-zero = deterministic), construct `*game.Game` with `(seed, opts)`, pass it to `play.New`, and print a `Seed: <N>` line at startup so users can reproduce a session. Update `cmd/play_test.go` to cover the flag.

## 10. Golden-Game Integration Tests and Smoke

- [x] 10.1 Add `internal/game/golden_test.go` realizing the design decision "Test strategy: state-machine unit + golden-game integration + manual smoke" — run a complete deterministic round (seed-pinned, four hands east-only, all bots) and capture the resulting event log into `testdata/game/golden/seed-<N>.json`. The test SHALL diff against the file and fail loudly on drift; `go test -update` SHALL regenerate the golden. Land at least one captured golden file under `testdata/game/golden/`.
- [x] 10.2 Run `just lint` and `just test` to verify the change is clean. Manual smoke: launch `mahjong play --seed 42` and `mahjong play --ascii --seed 42`, confirm a hand is dealt, you can move the cursor, discard, see bot discards appear in the correct per-seat zones, and quit cleanly with `q`.
