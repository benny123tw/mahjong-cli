## Why

The rules engine and `mahjong calc` CLI are complete and archived (see capability `hand-calculator`). The author's stated goal is to write a riichi mahjong game; a static calculator is a study tool but doesn't simulate play. The TUI play screen is the smallest surface that converts the project from "a calculator" to "a game in progress" and exercises the engine's interface in interactive use.

This change deliberately scopes to **layout and rendering only** — no game state, no opponent simulation, no engine wiring beyond projecting a `hand.Hand` value for display. The goals are: prove the layout works on the author's terminal at the chosen tile-rendering strategy; lock in the keybinding map before muscle memory forms; and learn bubbletea v2's MVU idioms on a small slice before adding game flow. Subsequent changes (`add-game-loop`, `add-trainer-aids`) build on this skeleton.

## What Changes

- Add `mahjong play [--ascii]` cobra subcommand that launches the TUI program
- Implement an `internal/play` package holding the bubbletea v2 program: model, Init, Update, View, and the keymap
- Render the play-screen layout at **fixed 80×24**:
  - Status line at top (round/honba/wall-count/scores — all hardcoded values for this change)
  - Toimen (opposite seat) tile backs as a horizontal row across the upper region
  - Kamicha (left) and Shimocha (right) tile backs as vertical strips
  - Centre pond rendering everyone's discards (hardcoded sample discards)
  - Dora indicator inset within the status region
  - Player's hand at the bottom with a cursor highlight on the focused tile
  - Action button row as a footer
- Implement two interchangeable tile renderers behind a shared interface:
  - **Unicode** renderer using mahjong glyphs from the U+1F000 block, appending the VS-15 text-variation selector (`︎`) to force monochrome presentation and stable cell width
  - **ASCII** renderer using boxed forms (`┌──┐│1m│└──┘`) selected via the `--ascii` flag
- Implement the keybinding map (keyboard only — no mouse this change):
  - `←` / `→` or `h` / `l`: move cursor across the player's hand
  - `1`–`9`: jump cursor to the nth tile in hand
  - `d` or Enter: discard the tile under the cursor (no-op visual feedback only — game-state wiring lands in `add-game-loop`)
  - `r`: declare riichi (no-op visual)
  - `t`: tsumo (no-op visual)
  - `p` / `c` / `k`: pon / chi / kan (no-op; rendered greyed in the action footer)
  - Space: pass / no-call (no-op)
  - `?`: machi / yaku peek (bound but no-op; trainer-aid behavior lands in `add-trainer-aids`)
  - `q` or Ctrl+C: quit
- Receive `tea.WindowSizeMsg` from program startup and on resize, and store width/height on the model. The View() method renders fixed 80×24 regardless of stored dimensions in this change, but the data is captured so reflow can be added in a future change without a model rewrite. If reported terminal dimensions are smaller than 80×24, render a "terminal too small (need 80×24)" notice in place of the layout
- Hardcode a known winning hand for the player's hand display (the chinitsu+toitoi+sanankou smoke-test hand `1m1m1m4m4m4m7m7m7m9m9m9m5m5m`) and hardcoded 13-tile-back counts for each opponent

## Non-Goals (optional)

- Game state — wall, dealing, draw/discard cycle, opponent agency, scoring updates (deferred to `add-game-loop`)
- AI opponents (deferred to a follow-up change after the game loop exists)
- Trainer aids — machi peek, furiten warning, illegal-call greying with engine-driven legality (deferred to `add-trainer-aids`)
- Dynamic reflow — compact mode for sub-80-wide terminals or wide-mode sidebar for larger terminals (deferred until friction emerges in real use; the model captures dimensions so this is purely a View() change later)
- Mouse support — keyboard-only this change; mouse is additive and can land later without breaking the existing keymap
- Automated tests for the View() output — TUI rendering is verified by manual smoke testing in this change; golden-frame tests come back when `add-game-loop` introduces real state worth fixing in tests
- Engine wiring beyond projection — this change does not call into `internal/riichi/` for shanten, machi, yaku, or score; it only renders a `hand.Hand` value passed in from a hardcoded fixture

## Capabilities

### New Capabilities

- `play-screen`: A bubbletea v2 TUI screen rendering a riichi mahjong play layout at fixed 80×24, with a keyboard cursor moving across the player's hand, two interchangeable tile-rendering backends (Unicode glyphs and ASCII boxes), a documented keybinding map (no-op for actions in this change), and graceful "terminal too small" handling. Game state, opponents, and engine analysis are explicitly out of scope and deferred to follow-up changes.

### Modified Capabilities

(none)

## Impact

- Affected specs: new capability `play-screen`
- Affected code:
  - New:
    - cmd/play.go
    - internal/play/play.go
    - internal/play/render.go
    - internal/play/keys.go
  - Modified:
    - cmd/root.go (register the `play` subcommand)
    - go.mod
    - go.sum
  - Removed: (none)
- Dependencies added:
  - charm.land/bubbletea/v2 (the v2 line moved its canonical module path to charm.land; this is the upstream-recommended import path, not github.com/charmbracelet/bubbletea/v2)
  - charm.land/lipgloss/v2 (lipgloss v2 also lives at charm.land/lipgloss/v2; the v1 path at github.com/charmbracelet/lipgloss is incompatible with bubbletea v2's ANSI helper versions)
