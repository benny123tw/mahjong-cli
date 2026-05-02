## Problem

The play screen does not render the human player's called melds (pon, chi, ankan, minkan, shouminkan). After a successful call the meld is tracked correctly in the engine — `g.Melds(HumanSeat)` returns the right list, `IsHandOpen` flips true, yaku detection accounts for the meld, and call-window legality checks work — but the TUI never displays it. The player sees only their concealed-hand row and has no way to tell:

- whether their hand is still concealed (gating riichi, menzen tsumo, ippatsu, pinfu),
- what tile was called from which opponent (affects which yaku still apply, what shape they're committed to),
- why a Ron button is greyed out when the post-call shape is yakuless.

A real-game manual playtest surfaced this: the player called a pon, later saw `[R]on` greyed in the call window, and could not diagnose the cause because the called meld was invisible.

## Root Cause

`renderHand` in `internal/play/play.go` reads only `m.Hand()` — which delegates to `g.Hand(HumanSeat)` and returns the **concealed** tile slice. The function never inspects `m.game.Melds(HumanSeat)`. The renderer was authored before kan/pon support landed and was never extended when the engine started tracking open melds. The render path:

```
View → renderLayout → renderHand → m.Hand() → g.Hand(seat) [concealed only]
```

never branches into the meld set.

## Proposed Solution

Extend `renderHand` to render the human's open melds to the right of the concealed-hand block, separated from the concealed row by a 2-tile-width gap, with each meld's tiles grouped together and a single-tile-width gap between adjacent melds.

Each meld SHALL carry a textual seat-source marker placed immediately before the meld's tiles, formatted as `[<seat-letter>]` where `<seat-letter>` is `E` / `S` / `W` / `N`. For ankan (concealed kan) the marker SHALL be `[A]` (ankan, no called-from seat) — visually distinguishing it from minkan/shouminkan which carry a real source seat. Examples (Unicode renderer, where each tile glyph is one cell wide):

```
<concealed 13 tiles>   [E] 5p 5p 5p     (pon called from East)
<concealed 10 tiles>   [A] 1m 1m 1m 1m  (ankan)
<concealed  4 tiles>   [E] 5p 5p 5p 5p  [W] 9m 9m 9m  [A] 1z 1z 1z 1z  (3 melds)
```

The seat-source marker uses ASCII brackets in BOTH renderers — no glyph rotation, no per-renderer divergence beyond what already exists for the tile glyphs themselves. This trades visual elegance for clarity and avoids inventing a second rendering path.

When the open-meld block plus concealed hand exceed the 80-column row width, the renderer SHALL wrap the meld block onto a second line directly below the hand row. In practice 13 concealed (~39 cells at Width=3) + 2-cell gap + 3 melds at 3 tiles each + markers (~36 cells) fits comfortably; the wrap branch covers the worst case (4 kans + 13 concealed) which is rare but possible.

When the human has no open melds, the function SHALL render the existing concealed-only output unchanged (zero behaviour change for purely-concealed games).

## Non-Goals

- Rendering opponent open melds beneath their seat labels — the four-quadrant 80×24 layout makes this a dedicated layout exercise; ships in a follow-up change.
- Glyph rotation for the called tile (the JP-client convention) — the `[E]`-prefix marker is good enough for v1 and avoids inventing a second tile-rendering path per renderer.
- Sorting open melds into the concealed-hand row — open melds stay separate, never interleaved with concealed tiles.
- New CLI flags or rendering-mode toggles.
- Changing `renderer.Tile` or the `Renderer` interface — the marker is plain text emitted around the existing tile output.
- Color-coding ankan vs minkan/shouminkan beyond the `[A]` vs `[seat-letter]` text difference.

## Success Criteria

- When the human player has at least one open meld, `View()` output contains the meld's tiles AND a `[<seat-letter>]` or `[A]` marker, separated from the concealed-hand row.
- When the human has zero open melds, the rendered output is byte-identical to the pre-change rendering (regression-safe).
- Ankan, minkan, pon, chi, and shouminkan all render with the correct tile count (4 / 4 / 3 / 3 / 4 respectively) and the correct seat marker.
- The render fits within 80 columns for any realistic hand (13 concealed + up to 3 melds without wrap; 4 melds wrap to a second line).
- `go test ./internal/play/` passes including new render tests asserting that planted open melds appear in the rendered output.

## Impact

- Affected code:
  - Modified: `internal/play/play.go` (new `renderOpenMelds` helper plus a `renderHand` extension that joins the meld block via `lipgloss.JoinHorizontal`; potentially a small line-wrap helper if the row exceeds 80 columns), `internal/play/play_test.go` (new tests asserting open-meld rendering and the no-meld regression-safety case)
  - New: none — all changes are extensions of existing files
  - Removed: none
