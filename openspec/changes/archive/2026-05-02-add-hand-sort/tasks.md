## 1. Engine: Human Hand Canonical Sort Invariant

- [x] 1.1 Implement the Human Hand Canonical Sort Invariant: add a `sortHumanHand()` helper in `internal/game/turn.go` that sorts a 13-tile slice in canonical tile-ID ascending order; export the canonical comparator if it's already useful elsewhere, otherwise keep it package-private.
- [x] 1.2 In `internal/game/turn.go`, identify every site where the human seat's concealed hand is mutated: initial deal, after-discard, after-call (pon/chi). At each site, invoke the sort helper before the state transition completes. Bot seats SHALL be skipped by checking `seat == HumanSeat`.
- [x] 1.3 Confirm the drawn-tile flow: when the human transitions into `AwaitingDiscard{Player: Human}`, the drawn 14th tile SHALL be appended at index 13 AFTER the sort runs on indices 0..12, so the drawn tile is preserved at the rightmost slot regardless of canonical position.
- [x] 1.4 Verify in `internal/game/turn_test.go` (or the relevant existing test file): write a test that drives the human through deal → draw → discard-the-drawn-tile and asserts the resulting 13-tile hand is sorted at every observable step (before draw and after discard).
- [x] 1.5 Add a test asserting that discarding a sorted-hand tile (e.g., index 5) produces a 13-tile hand still in canonical sort order on the next turn — the drawn tile correctly slots into its sorted position when the player discards a non-drawn tile.
- [x] 1.6 Add a test asserting bot hands are NOT sorted — drive a multi-turn game with a fixed seed, read a bot's hand mid-game, and verify the engine performs no ordering work for bot seats (the assertion can be a behavior contract test: confirm `Game.Hand(SeatEast)` after deal does NOT necessarily satisfy ascending-ID order, OR — easier — that no sort code path runs for non-human seats by inspecting the call site).

## 2. Render: Play Screen Layout drawn-tile gap

- [x] 2.1 Update the Play Screen Layout hand region: in `internal/play/play.go`, locate the hand-rendering code in `View()` and modify it so when the underlying state is `AwaitingDiscard{Player: Human}` AND the hand has 14 tiles, the renderer emits the leftmost 13 tiles densely concatenated, then exactly one tile-slot's worth of horizontal whitespace (using the active renderer's `Width()`), then the 14th tile.
- [x] 2.2 When the state is NOT `AwaitingDiscard{Human}` (i.e., `AwaitingClaims`, `AwaitingDraw`, `RoundOver`, etc.) and the hand has exactly 13 tiles, all 13 SHALL render densely with no gap. The drawn-tile separator SHALL only appear in `AwaitingDiscard{Human}`.
- [x] 2.3 In `internal/play/play_test.go`, add a test that constructs a `NewWithGame()` model, drives it into `AwaitingDiscard{Human}` with a known 14-tile hand, calls `View()`, and asserts the rendered hand string contains a single empty-tile-slot gap between the 13th and 14th tiles. Use `UnicodeRenderer{}` so the test runs against the production glyph path.
- [x] 2.4 Add a parallel test that drives the model into a non-`AwaitingDiscard` state with a 13-tile hand, calls `View()`, and asserts no gap appears in the rendered hand region.
- [x] 2.5 Verify cursor handling still works after the gap renders: write a test that moves the cursor to index 13 (the drawn tile) and asserts the cursor is highlighted on the 14th tile, not lost in the gap.

## 3. Golden Test Regeneration

- [x] 3.1 Run `go test ./internal/game/ -run TestGoldenSeed -update` to regenerate `testdata/game/golden/seed-42.json` because the dealing log now reflects sorted human hands. Inspect the diff to confirm the only difference is the human seat's hand ordering at deal/post-discard/post-call sites — bots' hands and all other event fields SHALL remain byte-identical.
- [x] 3.2 Re-run `go test ./internal/game/` without `-update` and confirm the golden test passes against the regenerated fixture.

## 4. Manual Smoke Test

- [x] 4.1 Build the binary (`go build -o bin/mahjong .`) and launch via `./bin/mahjong play --seed 42`. Visually confirm: (a) the dealt 13-tile hand is sorted M-then-P-then-S-then-honors; (b) on draw, the 14th tile appears at the rightmost slot with a visible gap before it; (c) discarding the drawn tile collapses the gap and leaves a still-sorted 13-tile hand for the next turn; (d) discarding a non-drawn tile re-sorts the remaining 13.

## 5. Verification

- [x] 5.1 Run `go test ./...` from the project root and confirm all tests pass — engine tests, play tests, and the regenerated golden fixture.
- [x] 5.2 Run `golangci-lint run ./...` and confirm 0 issues.
- [x] 5.3 Run `spectra validate add-hand-sort` and confirm the change validates.
