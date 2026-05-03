## Context

The mahjong engine already has a complete hand-end and payout pipeline:

- `internal/game/turn.go` transitions to `StateRoundOver{Outcome}` when a hand ends (ron / tsumo / ryuukyoku).
- `Outcome` is a sum type: `OutcomeRon{Winner, Loser, Result}`, `OutcomeTsumo{Winner, Result}`, `OutcomeRyuukyoku{TenpaiPlayers}`. `Result` carries `Award.Total`, `Award.Base`, the yaku list, han, and fu.
- `Match.AdvanceFromOutcome` applies `ComputePayouts(outcome, ctx)` deltas to per-seat scores and constructs the next hand's game.
- `internal/play/play.go` already detects `StateRoundOver` and shows a transient ack panel that displays per-seat point deltas; the panel waits for any key, then advances to the next hand or to standings.

The current ack-panel content is minimal — it shows the deltas and the transition reason ("ron from West", "ryuukyoku"). It does NOT reveal the four hands, the yaku list for a winner, the han/fu/base breakdown, or per-seat tenpai/noten on draws. The information exists on `Outcome.Result` (for wins) and on `Outcome.TenpaiPlayers` (for ryuukyoku); it is simply not rendered.

This change is therefore PURELY a play-screen renderer change: enrich the ack panel content with information already on the engine side. No new state, no new payout math, no new key handler — the existing "any key advances" mechanic stays intact.

Existing machinery to leverage:

- `Outcome.Result.YakuList` — winner's yaku breakdown (per-yaku name + han).
- `Outcome.Result.Han`, `Outcome.Result.Fu`, `Outcome.Result.Award.Base` — score components.
- `ComputePayouts(outcome, ctx) [4]int` — per-seat deltas for any outcome (already called by `Match.AdvanceFromOutcome`; the play layer can call it directly to display deltas without re-deriving).
- `OutcomeRyuukyoku.TenpaiPlayers []Seat` — direct lookup, no shanten re-check needed.
- `Game.Hand(seat)`, `Game.Melds(seat)` — face-up reveal data.
- `internal/play/play.go renderOpenMelds()` — meld rendering, currently HumanSeat-only; generalizing to take a seat parameter is the only structural code change.

## Goals / Non-Goals

**Goals:**

- Single panel renderer that handles ron, tsumo, chankan-ron (a sub-flavor of ron), and ryuukyoku.
- Per-seat hand reveal stays consistent with the existing concealed-hand renderer (same tile glyphs, same open-meld renderer extended to take a seat parameter).
- Read everything from `Outcome` and `Game` — no engine-side changes.
- Preserve the existing "any key advances" ack-panel mechanic; only the rendered content changes.

**Non-Goals:**

- Multi-winner support (double / triple ron). Engine emits a single winner today; if it later supports multi-winner, the panel must extend (deferred).
- Abortive draws (kyuushuu, suucha-riichi, suukaikan, suufon-renda) — out of this change's scope; the panel only fires for `OutcomeRon`, `OutcomeTsumo`, and `OutcomeRyuukyoku`.
- Hanchan progression — already implemented in `Match.AdvanceFromOutcome`; this change leaves it untouched.
- Animations or color flair beyond winner-row highlight + winning-tile highlight.
- Per-line fu derivation prose — fu total only.

## Decisions

### Read panel content from the captured outcome variant

Two alternatives were considered:

1. **Add a new `StateHandEnded` engine state** carrying the breakdown explicitly. This was the original plan before the existing `StateRoundOver` and `Outcome` types were verified.

2. **Read from the existing `OutcomeRon.Result` / `OutcomeTsumo.Result` / `OutcomeRyuukyoku.TenpaiPlayers` directly** in the play-screen renderer. `Result` already carries the yaku list, han, fu, and award; `TenpaiPlayers` already lists who was tenpai; `ComputePayouts` derives the deltas.

Decision: **alternative 2**. The engine has already designed exactly what the panel needs; introducing a parallel state would be redundant. The renderer reads `Outcome` straight off `StateRoundOver`.

### Rename `renderOpenMelds` to `renderOpenMeldsForSeat`

Current signature: `(m Model) renderOpenMelds() string` reads `m.game.Melds(HumanSeat)`.

New signature: `(m Model) renderOpenMeldsForSeat(seat game.Seat) string`. The non-panel call site (`renderHand`) updates to call `renderOpenMeldsForSeat(HumanSeat)`. The panel renderer calls it for each of the 4 seats. This is a low-risk parameter extraction — behavior for the existing path stays byte-identical.

### Create `renderEndPanel` with header, seat rows, breakdown, footer

The 80×24 budget breaks down as:

- 1 header line: kind tag (e.g., `RON — South wins on 8m from West`, `TSUMO — South wins`, `RYUUKYOKU`, `CHANKAN RON — South wins on 4z from East`).
- 4 seat rows, each 1 line tall (Unicode renderer `Lines()=1`) — seat label (E/S/W/N), seat wind tag, concealed hand glyphs, then open melds via `renderOpenMeldsForSeat`. With Unicode `Width()=3` and 13 concealed: ~3 cols seat label + ~6 cols wind tag + 39 cols concealed + 2-tile gap + open melds → fits within 80 if open-melds total ≤ ~28 cols (typical 3-meld case). For overflow, melds wrap below the seat row, same as the existing `renderHand` wrap path.
- 1 spacer line.
- Win panel: 1 line for the compact yaku list (`Yaku: Riichi 1 · Pinfu 1 · Tanyao 1 · Tsumo 1`), 1 line for `Han N · Fu M · Base K`, 1 line for the per-seat deltas (`South +8000 · East -8000 · West 0 · North 0`).
- Ryuukyoku panel: per-seat `tenpai`/`noten` labels appended inline to seat rows (no extra line), then 1 line for the per-seat deltas.
- 1 footer line: `[Any key — Continue]`.

For ASCII renderer (`Lines()=3`) the panel drops to 2 rows worst case for the seat-row block; the panel still fits because there is no live keyboard cursor, status bar, or pond zone competing for vertical space when the ack panel is active.

Long yaku lists wrap by emitting up to 3 lines for the yaku block (single line with overflow → second line indented under `Yaku:`). Realistic upper bound: 13 yaku names + 13 han values fit in 2 lines comfortably.

### Winner-row highlight + winning-tile highlight

Winner row gets a `[W] ` prefix in front of the seat label (4 chars) styled bold. The winning tile within the winner's concealed hand is also styled bold via the existing `focusedTileStyle` (re-used; semantically distinct purpose, but visually identical and saves a new style entry).

For ron, the winning tile is `Outcome.Result.WinningTile` (already on `Result`); for tsumo, the same. The renderer locates the first occurrence of that tile-ID in the winner's hand and applies the highlight to that index; ties are broken by left-most.

### Switch on the captured outcome variant inside `renderEndPanel`

Single switch on `outcome := m.pendingAck.Outcome.(type)` in the panel's top-level `render(m Model) string`:

- `OutcomeRon` → `renderWinPanel(m, o, isChankan: m.pendingAck.IsChankan)`. The `IsChankan` flag already exists on the ack-panel state to distinguish chankan-ron from regular ron.
- `OutcomeTsumo` → `renderWinPanel(m, o, isChankan: false)`.
- `OutcomeRyuukyoku` → `renderRyuukyokuPanel(m, o)`.

Default branch: leave the existing minimal ack panel as-is for any future outcome variant the engine might add (graceful degradation).

## Risks / Trade-offs

- [Risk] The `Result.YakuList` field is the format expected by the renderer (slice of `{Name string; Han int}` or similar). If the actual type uses a different shape (e.g., bitmask, enum + lookup table), the renderer will need a small adapter. → Mitigation: verify the `Result` shape during apply Stage 1 (read `internal/riichi/score/score.go` and the yaku evaluator return type); if it's a bitmask, add a tiny `yakuListString` formatter in the renderer.

- [Risk] Open-meld layout can wrap to a second row (wide hand case), pushing the panel past 24 lines if all 4 seats wrap. → Mitigation: the render loop falls back to a `…` truncation indicator on seat rows that overflow horizontally rather than wrapping vertically when the panel is active. Test fixture for the worst case (4 ankans on the human + standard hands on others) is included.

- [Risk] `Result.WinningTile` may not exist as a named field on the score result (the engine may carry it on the outcome variant instead, e.g., `OutcomeTsumo.WinningTile`). → Mitigation: read `internal/game/state.go` during apply Stage 1 to confirm; the Outcome variants are the more likely source.

- [Trade-off] Reusing `focusedTileStyle` for the winning-tile highlight conflates two semantics (cursor vs. winner). The visual style is identical (bold), so the rendered output looks correct, but a future styling change to the cursor would unintentionally also restyle winners. → Acceptable for v1; if styling diverges later, introduce `winningTileStyle` then.

- [Trade-off] Reading `Outcome` and re-calling `ComputePayouts` from the play layer means the deltas are computed twice per hand-end (once by `Match.AdvanceFromOutcome`, once by the panel). The function is pure and cheap; the duplication keeps the play layer self-contained without snapshotting per-seat scores before/after the match transition. Acceptable.
