## Why

The user is a TW-mahjong player learning JP riichi (a side project, single-player against bots). The play screen has the game-rules pieces in place — wait calculation (`hand.Machi`), furiten detection (`g.IsFuriten`), call legality (`game.CanPon/CanChi/CanKan`) — but only one of the three TUI affordances that surface those mechanics is wired through:

- **`?` Peek key** is listed in `FooterKeys` as greyed and unbound. The Model already caches `peekMachi` on every refresh; the cache is just never displayed.
- **Furiten warning** appears in the call window via the Ron button's `(furiten)` suffix, but during `AwaitingDiscard{Human}` the player has no visible signal that ron is locked out.
- **Illegal-call greying** is already implemented in `RenderCallFooter` (greyed buttons via `greyedKeyStyle` + `(furiten)` suffix); no work needed.

This change finishes the first two so a learner gets the same wait-shape and furiten visibility that online clients (Tenhou / Mahjong Soul) provide — without changing engine behaviour, scoring, or bot decisions.

## What Changes

- The Model SHALL gain a `peekVisible bool` field, defaulting to `false`. Pressing `?` (KeyPress code `'?'`) SHALL toggle `peekVisible`. The toggle is a TUI-only state mutation and SHALL NOT call any `m.game.Step` or change game state.
- `peekVisible` SHALL be reset to `false` whenever the Model's cached `peekShanten` is cleared (the existing `m.peekShanten = peekUnknown` / `m.peekMachi = nil` sites in `handleDiscard`, the bot-tick handler, and the transition-pending handler), so the peek hides automatically when the next state begins.
- `renderFooter()` SHALL append a "Wait: <ids>" line below the action keys when (a) `peekVisible` is true, (b) `peekShanten == 0`, AND (c) `len(peekMachi) > 0`. The wait list SHALL render each tile ID via `tile.Tile{ID: id}.String()` separated by single spaces (e.g., "Wait: 4m 7m"). Empty / non-tenpai peeks SHALL render the line as "Wait: (not tenpai)" so the toggle still gives feedback.
- The `?` entry in `FooterKeys` SHALL change `Greyed: true` to `Greyed: false` (it is now a live binding).
- `renderFooter()` SHALL append a "[FURITEN]" badge in red (via a new `furitenBadgeStyle` lipgloss style — `Foreground` set to a red tone) when (a) the current state is `StateAwaitingDiscard{Player: HumanSeat}` OR `StateAwaitingDraw{Player: HumanSeat}` (i.e., the human's turn cycle outside of the call window — the call window already labels the Ron button), (b) the human's hand is at tenpai (`hand.Shanten(humanHand) == 0`), AND (c) `g.IsFuriten(HumanSeat)` returns true. ASCII renderer mode SHALL render the badge as `(furiten)` with no color (matches the existing call-footer convention).
- New unit tests in `internal/play/play_test.go` SHALL cover: peek toggle on/off, peek auto-clears on discard, peek shows the right wait IDs for a tenpai shape, furiten badge appears when human is tenpai + furiten, furiten badge does NOT appear when human is non-tenpai or not-furiten.

## Non-Goals

- Highlight wait tiles inside the human's actual hand row (would require per-tile coloring inside the hand renderer; a larger render-pipeline change).
- Machi peek for non-tenpai shapes (would need to enumerate post-discard waits per candidate discard — a different feature).
- Machi peek for opponents / bots (information exposure that breaks the single-player learning model).
- Visual coloring of dora / aka-dora tiles in the hand row (a separate visual-polish change).
- Tutorial overlays, first-launch onboarding, or in-game help text beyond the existing footer hints.
- A new `Game.Machi(seat)` engine accessor — the Model already computes the human's Machi locally via `hand.Machi(hand.Hand{Concealed: humanHand})`.
- Changes to `RenderCallFooter` — the call-window greying and `(furiten)` suffix on the Ron button are already in place.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `play-screen`: `Action Footer Rendering` (or the equivalent existing requirement that documents footer behaviour) updated to add the `?` Peek binding behaviour, the wait-line rendering, and the furiten-badge rendering during the human's turn cycle.

## Impact

- Affected specs: `play-screen` (modified — action-footer behaviour for `?` peek, wait line, furiten badge)
- Affected code:
  - Modified: `internal/play/play.go` (peekVisible field, `?` key handler, peek auto-clear at the existing reset sites, `renderFooter()` extension for wait line and furiten badge), `internal/play/keys.go` (FooterKeys: `?` Peek entry flips Greyed false), `internal/play/play_test.go` (new tests for peek toggle / auto-clear / furiten badge)
  - New: none
  - Removed: none
