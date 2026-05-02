## 1. Project setup

- [x] 1.1 Add bubbletea v2 (`charm.land/bubbletea/v2` — the canonical v2 module path) and `github.com/charmbracelet/lipgloss` as dependencies, per the Bubbletea v2 with the MVU split — model in `internal/play`, command in `cmd/play.go` decision
- [x] 1.2 Establish the internal/play package layout — `internal/play/play.go` for the model/Init/Update/View, `internal/play/render.go` for the renderer interface and implementations, `internal/play/keys.go` for the keymap — model in internal/play, command in cmd/play.go split mirroring the existing rules-engine adapter pattern

## 2. Tile renderer

- [x] 2.1 Implement the Tile Rendering Strategy — the Renderer interface with two implementations, selected once at startup, plus the Unicode renderer producing U+1F000-block glyphs with U+FE0E (VS-15) appended for monochrome cell-width stability
- [x] 2.2 Implement the ASCII renderer using boxed 4-column × 3-row tile forms (e.g., `┌──┐│1m│└──┘`)

## 3. Model and update

- [x] 3.1 Implement the model struct holding cursor position, terminal width and height, renderer choice, and the Hardcoded Fixture For Display — the Hardcoded fixture is the chinitsu+toitoi+sanankou smoke-test hand `1m1m1m4m4m4m7m7m7m9m9m9m5m5m`
- [x] 3.2 Implement the Window Size Captured On Model handling — receive `tea.WindowSizeMsg` in Update and update model width/height per the Fixed 80×24 layout, dimensions stored but ignored by View() decision
- [x] 3.3 Implement the Keybinding Map — Keybinding map — keyboard only, action keys bound but inert: cursor movement on `←`/`→`/`h`/`l`/`1`–`9`; action keys (`d`, Enter, `r`, `t`, `p`, `c`, `k`, Space, `?`) produce only visual acknowledgement; `q` and Ctrl+C exit cleanly

## 4. View

- [x] 4.1 Implement the Play Screen Layout View — render the six fixed regions (status line, toimen horizontal tile-back row, kamicha and shimocha vertical tile-back strips, centre discard pond, dora indicator, player's hand at bottom with cursor highlight, action footer) at fixed 80×24 with lipgloss color, centered when the terminal is larger
- [x] 4.2 Implement the "terminal too small (need 80×24)" notice path when reported terminal dimensions are below 80 columns or 24 rows
- [x] 4.3 Wire the player's hand fixture, 13 opponent tile-backs per opposite seat, sample dummy discards in the centre pond, and fixed status-line constants (round/honba/wall/scores) into the View per Hardcoded Fixture For Display

## 5. CLI integration

- [x] 5.1 Add `cmd/play.go` implementing the Play Subcommand Launch — register a cobra `play` subcommand on the root command with the `--ascii` boolean flag, parse it, construct the play model with the appropriate renderer, and start the bubbletea program

## 6. Verification

- [x] 6.1 Manual smoke-test per the Test strategy: manual smoke-test only — launch `mahjong play` and `mahjong play --ascii` in the author's terminal at 80×24, at a larger size (≥120×40 to verify centering), and at sizes below 80 columns and below 24 rows (to verify the too-small notice); confirm cursor movement with arrows / h / l / number keys and that `q` and Ctrl+C exit cleanly
