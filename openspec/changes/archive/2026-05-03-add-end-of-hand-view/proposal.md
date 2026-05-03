## Why

The existing end-of-hand ack panel (see `play-screen` requirement `End-of-Hand Acknowledgement`) shows the per-seat point deltas and waits for a keypress before advancing to the next hand, but it does NOT reveal opponent hands, the winner's yaku list, or the han/fu/base breakdown. For a player who is learning JP riichi (which is the entire point of this CLI), the deltas alone are insufficient — without seeing the agari hand and the yaku that scored it, there is no learning signal. Similarly on ryuukyoku: the player sees a delta but cannot see who was tenpai, what shape they were waiting for, or why the payments landed where they did.

The engine already computes everything needed: `ComputePayouts` returns deltas, `OutcomeRon{Winner, Loser, Result}` carries the yaku evaluation result on wins, `OutcomeRyuukyoku{TenpaiPlayers}` carries the tenpai set on draws, `Game.Hand(seat)` and `Game.Melds(seat)` expose every seat's hand and called melds. The change is entirely in the play-screen renderer: enrich the ack panel with the reveal + breakdown information that is already on the engine side waiting to be displayed.

## What Changes

- The end-of-hand ack panel SHALL render all four seats' concealed hands face-up (no `Back()` glyph), each labeled with seat letter (E/S/W/N) and seat wind, plus open melds via the same `renderOpenMelds` machinery used for the human (extended to take a seat parameter).
- For ron / tsumo wins (`OutcomeRon`, `OutcomeTsumo`):
  - The winner's row SHALL be visually marked (color-bold or `[W]` prefix) and the winning tile within the winner's concealed hand SHALL be highlighted.
  - For ron: the dealt-in seat (loser) SHALL be identified next to the panel header.
  - For chankan-ron: the panel header SHALL include `chankan`.
  - Below the four hands, a breakdown SHALL list each yaku name with its han value (compact: `Riichi 1 · Tsumo 1 · Pinfu 1`), then total han, fu, base score, and the per-seat payout deltas.
- For ryuukyoku (`OutcomeRyuukyoku`):
  - Each seat row SHALL get a `tenpai` or `noten` tag based on whether that seat appears in `TenpaiPlayers`.
  - Below the four hands, the panel SHALL show the noten-penalty payments per seat (the deltas are already produced by `ComputePayouts`).
- The footer key strip SHALL change to `Any key — Continue` while the ack panel is active. Existing behavior of "any key advances to the next hand or to standings" is preserved; only the label changes for clarity.
- The `renderOpenMelds` function SHALL be generalized: a seat parameter replaces the current hard-coded `HumanSeat`. The non-panel call site (`renderHand`) updates to pass `HumanSeat` explicitly. This is a pure parameter extraction — behavior for the existing path is byte-identical.

## Non-Goals (optional)

- Engine state or payout changes — `ComputePayouts`, `OutcomeRon/Tsumo/Ryuukyoku`, and `Match.AdvanceFromOutcome` already produce everything the panel needs.
- Changing the existing ack-panel mechanics: keypress still advances to the next hand (or standings on hanchan-end / tobi). This change only enriches the panel content.
- Per-line fu derivation prose (e.g., "20 base + 2 minkou of 1z + ..."). Show the fu total only.
- Animations, color flair beyond seat-highlight and winning-tile highlight, or per-stage payment reveal.
- Multi-winner support (double / triple ron). Engine emits a single winner; if it later supports multi-winner, the panel will need to extend.
- Abortive draws (kyuushuu, suucha-riichi, suukaikan, suufon-renda). Only `OutcomeRon`, `OutcomeTsumo`, and `OutcomeRyuukyoku` trigger the rich panel; if the engine ever introduces a fourth outcome variant, the panel will need a fall-through.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `play-screen`: Enriches the existing `End-of-Hand Acknowledgement` requirement to add the 4-hand reveal, yaku/fu/payout breakdown for wins, and tenpai/noten labels for ryuukyoku.

## Impact

- Affected specs:
  - `openspec/specs/play-screen/spec.md` (modified): `End-of-Hand Acknowledgement` requirement gets a richer panel-content contract.
- Affected code:
  - New: `internal/play/endpanel.go` (panel renderer — reveal layout + win/ryuukyoku breakdown formatters).
  - Modified: `internal/play/play.go` (call the new renderer when ack-panel is active; generalize `renderOpenMelds` to take a seat parameter).
  - Modified: `internal/play/play_test.go` (panel rendering tests for ron, tsumo, and ryuukyoku flavors).
