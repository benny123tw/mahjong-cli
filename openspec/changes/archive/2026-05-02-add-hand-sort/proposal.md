## Summary

Auto-sort the human player's hand in canonical riichi order while keeping the just-drawn 14th tile separated at the right end so the player can tell what they drew.

## Motivation

The skeleton change shipped with the TUI displaying the player's hand in arrival order — whatever sequence the wall happened to deal. Real riichi UI always sorts the hand by suit (M / P / S / honors) so the player can recognize sequences and pairs at a glance. The author is a Taiwanese-mahjong player learning Japanese riichi; reading shape from a sorted hand is the primary cognitive skill the game wants to train, and an unsorted hand actively works against learning. Keeping the drawn tile separated (Mode B from the discussion) preserves the "what did I just draw?" affordance that real physical mahjong gives via the right-end-of-the-rack convention.

## Proposed Solution

Introduce a stable invariant for the human player's hand: every time the underlying 13-tile concealed hand changes (deal, after-discard, after-call), it SHALL be re-sorted by canonical tile order (M1..M9, P1..P9, S1..S9, EastWind, SouthWind, WestWind, NorthWind, Haku, Hatsu, Chun). The just-drawn 14th tile SHALL be appended to the sorted hand AFTER the sort runs, so it always lives at the rightmost slot during `AwaitingDiscard{Human}` state.

Rendering: the play screen renders the leftmost 13 tiles densely (existing behaviour) and inserts a single visible gap (one tile-slot's worth of horizontal padding, or a thin separator glyph in Unicode mode) before the 14th drawn tile. After the player discards either the drawn tile or any sorted-hand tile, the resulting 13 tiles are re-sorted and the gap goes away (next turn starts with a homogeneous sorted hand).

Cursor handling: the existing cursor maps to a hand index 0..n-1 of `Game.Hand(seat)`. After sorting, indices map to different tiles than before. We accept this — the cursor's position-on-screen stays meaningful (it points at "tile under cursor"), and the player normally moves the cursor via arrow keys interactively rather than memorizing index numbers.

Scope is limited to the human seat's hand. Bots' hands are never displayed; their algorithmic decisions don't need a sorted view. Opponent ponds are NOT re-sorted (riichi convention is chronological discard order, which we already render correctly).

## Non-Goals

- Sorting opponent ponds (chronological order is the riichi convention and is already correct).
- Reordering called melds (open chi/pon meldsd have a fixed left-to-right discard-source order; reordering them would lose information about who fed the call).
- Animations or transitions between sort states.
- A user-toggleable sort mode (always-sort is the contract; if a future user wants raw arrival order, that ships in a separate change).
- Sorting the bot seats' internal hands. Bot decision logic is order-independent and adding a sort would be a wasted side effect.

## Alternatives Considered

- **Mode A: always sort the full 14-tile hand including the drawn tile.** Rejected because it loses the "what did I just draw?" affordance. A learner can no longer tell which tile completed the shape they're looking at.
- **Sort on every render, not on every mutation.** Rejected because the cursor maps to an index in the underlying hand and re-sorting on every render means the underlying order does not match the rendered order, which silently breaks discard-by-index.
- **User-toggleable sort.** Out of scope per Non-Goals — adds two ways the hand can look at every moment for marginal value.

## Impact

- Affected specs: modified capability `play-screen` (player's hand rendering); modified capability `game-loop` (the sort happens at the engine layer so the engine's `Game.Hand(seat)` view is already-sorted for display).
- Affected code:
  - Modified:
    - internal/game/turn.go (sort the human seat's hand on deal, after discard, after call)
    - internal/play/play.go (render a single visual gap before the 14th tile when state is `AwaitingDiscard{Human}`)
    - internal/game/turn_test.go OR internal/game/state_test.go (extend with sorted-hand assertions)
    - internal/play/play_test.go (extend with drawn-tile-gap rendering assertions)
    - internal/game/golden_test.go (golden file regenerates because sorted hands change the dealing log; run with -update once)
  - New:
    - (none — the change is small enough to live entirely in existing files)
  - Removed:
    - (none)
