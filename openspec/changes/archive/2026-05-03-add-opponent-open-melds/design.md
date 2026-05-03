## Context

The play screen renders three opponent zones in the four-quadrant layout (`internal/play/play.go`):

- `renderToimenRow()` — top zone for North. Renders `label · score`, then a 13-tile face-down back-row, then the discard pond. Full ~80-column width (centered).
- `renderKamichaColumn()` — left middle zone for East. Renders `label · score` then the discard pond. ~20 columns wide (constrained by `lipgloss.NewStyle().Width(20)` in `renderMidRow`).
- `renderShimochaColumn()` — right middle zone for West. Same shape as Kamicha. ~20 columns wide.

The human's open melds already render to the right of the concealed-hand row (`add-open-meld-display`); opponents have NO meld rendering anywhere on the live play screen. The end-of-hand reveal panel (`add-end-of-hand-view`) shows them but only after the hand terminates.

`renderOpenMeldsForSeat(seat game.Seat) string` was generalized in `add-end-of-hand-view` exactly so it could be reused per-seat. It walks `m.game.Melds(seat)` and emits the per-meld block with seat-source markers attached to the called tile (`[E]`/`[S]`/`[W]`/`[N]`) or `[A]` for ankan. Returns "" for a seat with zero melds.

This change wires that helper into the three opponent zone renderers.

## Goals / Non-Goals

**Goals:**

- Render opponent open melds inline within each opponent zone, between the seat-label header and the discard pond / face-down hand row.
- Zero-meld case stays byte-identical to the pre-change rendering for that zone.
- Wide-meld case (block exceeds zone width) wraps to a second line within the zone budget.
- Pathological case (block exceeds two lines of zone width) truncates with a `+K more` suffix.
- Stay within the existing 80×24 budget. No layout reflow.

**Non-Goals:**

- Yakuhai-meld highlighting (color, bold) — danger UX, deferred.
- Reconciling the face-down hand-glyph row to true post-call concealed hand size — deferred.
- ASCII renderer compact-mode beyond what `renderOpenMeldsForSeat` already produces.
- Three-or-more-line meld stacking. Cap at two lines; truncate beyond.
- Touching the human's per-zone rendering (their melds render via `renderHand` and that path is unchanged).

## Decisions

### Insert the meld block between the seat label and the pond

Each opponent zone currently follows the pattern `label · score \n [back-row \n] pond`. The new pattern becomes `label · score \n [meld-block \n] [back-row \n] pond`, with the meld block omitted entirely when the seat has zero open melds.

Two alternatives were considered:

1. **Header-adjacent** (above back-row / pond): keeps melds visually associated with the seat's identity row. The meld content and the seat label are both "what is this player?" data.
2. **Pond-adjacent** (below back-row, above pond, or interleaved with pond): groups melds with other "what they did" info.

Decision: **alternative 1** (header-adjacent). The seat label and melds together form the "this player's identity + what they have built so far" header for each zone. Discards are a chronological log; melds are static state. Putting them with the label keeps the static information together at the top.

### Wrap to 2 lines and truncate with `+K more`

The Kamicha and Shimocha columns are 20 cells wide; Toimen is ~80. `renderOpenMeldsForSeat` produces a single-line horizontal block. The wiring needs three branches:

1. **`lipgloss.Width(meldBlock) <= zoneWidth`** → render the block as-is on one line.
2. **Block width exceeds zoneWidth, but the seat has ≤ 2 melds total OR the block fits in 2× zoneWidth** → wrap by re-rendering one meld per line via `lipgloss.JoinVertical`. This wraps the meld block to 2 lines when a multi-meld block overflows.
3. **Even 2 lines do not fit (extreme case: 4 ankans on Kamicha, ~64 cells of meld content vs. 20-cell zone)** → render the first N melds that fit in 2 × zoneWidth, then append a styled `+K more` suffix where K is the remaining count.

Toimen's wide budget (80 cells) means every realistic meld count fits on one line. The wrap and truncation paths exist for Kamicha/Shimocha primarily.

### `+K more` suffix on a third line via lipgloss.JoinVertical

Two alternatives:

1. **Render first N melds on line 1, render `+K more` on line 2.** Visually clean separator.
2. **Render first N melds and `+K more` together on one line.** More compact, but `+K more` competes for the same 20-cell budget as the meld content.

Decision: **alternative 1**. Two-line cap + a dedicated `+K more` row keeps the wrap-vs-truncate transition visually distinct. The `+K more` suffix is rendered with `labelStyle` (subdued) so it does not compete visually with the actual meld content.

The cutoff for N: greedily fit melds left-to-right until adding the next meld would push the rendered width past `zoneWidth`. Use `lipgloss.Width` to measure; do not assume a fixed character count per meld (Unicode and ASCII renderers differ, and meld widths vary by kind).

### Human seat does NOT render melds in the four-quadrant layout

The human's seat does not get a per-zone meld render. Reasons:

- The human's melds already render to the right of the concealed-hand row in `renderHand` — that location is more visible during play (the human looks at their own hand row to make decisions, not at the centre quadrants).
- The human's "zone" in the four-quadrant layout is the bottom hand row — there is no symmetric `renderHumanColumn()` to extend.

This decision keeps the change small and avoids double-rendering the human's melds in two locations.

## Risks / Trade-offs

- [Risk] The 20-cell zone-width assumption depends on the existing `lipgloss.NewStyle().Width(20)` wrappers in `renderMidRow`. If the layout is later resized (e.g., 22 cells per side), the wrap/truncate thresholds need updating. → Mitigation: extract the zone widths as named constants (`kamichaZoneWidth`, `shimochaZoneWidth`) so changing the layout updates the meld renderer in lockstep.

- [Risk] Truncation with `+K more` may hide critical information (e.g., the last meld is an ankan of 1z that would tip off a yakuhai threat, but it gets truncated). → Mitigation: the end-of-hand reveal panel always shows everything; the player's authoritative view is at hand-end. During live play, truncation is accepted in exchange for layout stability. This trade-off is documented in the proposal.

- [Risk] Adding 1-2 lines per opponent zone may push the total layout past 24 rows when all three opponents have many melds. The current layout is tight (~24 rows used); two extra lines on each of Kamicha/Shimocha (in parallel via `JoinHorizontal`) consumes 2 rows total because they render side-by-side. Toimen's extra line adds 1 row at the top. Worst case: 3 extra rows. → Mitigation: verify the row budget during implementation. If it overflows, the wrap branch must be capped at 1 line for Kamicha/Shimocha (forcing earlier truncation).

- [Trade-off] The truncation form uses a styled `+K more` rather than a partial-tile glyph. The trade-off is verbose-but-readable vs. compact-but-cryptic. The current choice favors readability.

- [Trade-off] Refactoring all three zone renderers in lockstep means a single regression in the meld-insertion logic affects three zones. Mitigation: extract a single `renderOpponentMelds(seat, zoneWidth)` helper that all three zones call. Tests parameterize over the three opponent seats.
