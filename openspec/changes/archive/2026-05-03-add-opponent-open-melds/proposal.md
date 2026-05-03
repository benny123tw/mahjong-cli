## Why

The end-of-hand reveal panel (just landed via `add-end-of-hand-view`) shows every seat's open melds at hand termination. During live play, however, opponents' open melds are still completely invisible — only the human's melds render (via `renderHand`). A bot may have called a pon of 5p from East and an ankan of 1z, and the human has no way to see that until the hand ends. They cannot reason about danger (open hand → likely yakuhai-driven? which suit is being collected?), cannot identify yakuhai threats, and cannot plan defensively. The reveal panel makes the gap obvious by contrast: the same information that closes the play loop is unavailable while decisions are being made.

Closing this gap completes the symmetry: open-meld visibility becomes consistent across the live play layout and the end-of-hand panel. The infrastructure is already in place — `renderOpenMeldsForSeat(seat)` was generalized in the previous change exactly so it could be reused per-seat — so this is purely a rendering wiring task in the play-screen layout.

## What Changes

- Each opponent zone (Kamicha/East, Toimen/North, Shimocha/West) SHALL render that seat's open melds inline within the zone, using the same `renderOpenMeldsForSeat(seat)` function used by `renderHand` and the end-of-hand reveal panel.
- The meld block SHALL render directly under the existing seat-label header line and above the face-down hand-glyph row and the discard pond.
- When a seat has zero open melds, the meld region SHALL contribute nothing to the rendered output for that zone — no extra blank line, no `(none)` label. Layout for the zero-meld case stays byte-identical to the pre-change rendering.
- When the meld block exceeds the zone's column budget, the renderer SHALL wrap to a second line within the zone via `lipgloss.JoinVertical` (the same wrap pattern `renderHand` uses for the human's wide-hand case).
- For pathological cases (4 ankans on a single opponent — extreme but legal with 4 declared kans), the renderer SHALL fall back to a compact form: render only the first N melds that fit in the zone width, then append a `+K more` suffix where K is the count of unrendered melds. The end-of-hand reveal panel still shows everything (no truncation there), so the player retains an authoritative view at hand-end.
- The human seat's per-zone rendering in the four-quadrant layout SHALL NOT render melds in this new opponent-style location. The human's melds already render to the right of the concealed-hand row (`add-open-meld-display`), and that location is more visible. Avoid double-rendering.

## Non-Goals (optional)

- Reflowing the four-quadrant layout to give opponent zones more horizontal space. Stay within the existing 80×24 budget.
- Color or extra emphasis on yakuhai melds (e.g., highlighting a Haku pon as "yakuhai threat"). That is danger-aware UX and belongs in a smart-AI / trainer-aid follow-up.
- Reconciling the face-down hand-glyph row to reflect post-call concealed-hand size. Today the layout always shows 13 face-down tiles per opponent regardless of how many they have actually called; that is a separate concern (depends on whether the engine surfaces per-seat post-call hand size to the play layer).
- ASCII-renderer-specific compact-mode beyond what `renderOpenMeldsForSeat` already produces. The existing markers (`[E]`, `[S]`, `[W]`, `[N]`, `[A]`) work the same in both renderers.
- Multi-line meld stacking that grows the opponent zone vertically beyond two lines. If a single opponent has so many melds that even a 2-line block does not fit, the truncation-with-`+K more` fallback applies.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `play-screen`: Adds opponent open-meld rendering inside each opponent zone of the four-quadrant layout, with wrap and truncation handling.

## Impact

- Affected specs:
  - `openspec/specs/play-screen/spec.md` (modified): the `Play Screen Layout` requirement gains opponent open-meld rendering rules and the `Open Meld Display For Human Player` requirement is unchanged (the human-side path stays as-is).
- Affected code:
  - Modified: `internal/play/play.go` — per-opponent zone-render code (each opponent zone gains a meld-block insertion between the seat-label header and the face-down hand row), plus a small helper for the truncation-with-`+K more` fallback.
  - Modified: `internal/play/play_test.go` — golden-style tests asserting opponent-zone output contains expected meld glyphs and seat-source markers, parameterized over 0/1/2/3-meld counts and the truncation-fallback case.
