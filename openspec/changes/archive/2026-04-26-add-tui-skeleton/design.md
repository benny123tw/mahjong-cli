## Context

The rules engine and `mahjong calc` CLI are archived (capability `hand-calculator`). The next milestone is interactive play, which requires a terminal UI. This change is the first wave: it builds the visual skeleton — layout, tile rendering, keybindings, window-size handling — without any game state, opponents, or engine analysis calls. It exists so two big risks can be retired before they tangle with game-state work:

1. **Tile rendering uncertainty.** Unicode mahjong glyphs (U+1F000–U+1F02F) render with inconsistent cell width across terminals, so the project commits to a Unicode-primary / ASCII-fallback strategy that needs to be proven on the author's actual terminal before we build on top of it.
2. **Bubbletea v2 unfamiliarity.** Bubbletea v2 is the project's MVU framework; the author hasn't used it. Doing the visual skeleton on a hardcoded fixture is the cheapest way to learn the idioms before adding game logic on top.

The architectural rule established in `add-hand-calculator` — engine code has zero UI dependencies — is preserved by directing the dependency the safe way: `internal/play` depends on `internal/riichi/hand` (the `hand.Hand` type) but the engine never imports `internal/play`.

## Goals / Non-Goals

**Goals:**

- Lock in the play-screen layout at 80×24 in the author's terminal, in both Unicode and ASCII rendering modes
- Lock in the keybinding map for the player's hand and action footer before muscle memory forms
- Establish the bubbletea v2 program structure (model / Init / Update / View / keymap split) that subsequent changes will extend with game state
- Capture window dimensions on the model from day one so reflow becomes a View() change later, not a model rewrite

**Non-Goals:**

- Game state (wall, dealing, draw/discard cycle, opponent simulation) — `add-game-loop`
- AI opponents — follow-up after `add-game-loop`
- Trainer aids (machi peek, furiten warnings, legality-aware action greying) — `add-trainer-aids`
- Engine analysis calls — this change projects a `hand.Hand` value for display only
- Dynamic reflow at any size — fixed 80×24, "too small" notice below that, centered if larger
- Mouse support — keyboard-only; mouse is additive and lands later if friction emerges
- Automated tests of View() output — manual smoke-testing only this change; golden-frame tests come with `add-game-loop`

## Decisions

### Bubbletea v2 with the MVU split — model in `internal/play`, command in `cmd/play.go`

`cmd/play.go` is a thin cobra adapter that parses flags and constructs an `internal/play.Program`. The play program lives in `internal/play` with the MVU surface split across files:

```
cmd/play.go              — cobra subcommand, flag parsing, program startup
internal/play/play.go    — Model struct, Init, Update, View
internal/play/render.go  — Renderer interface + Unicode and ASCII implementations
internal/play/keys.go    — Keymap (key.Binding values)
```

This mirrors the `cmd/calc.go` → `internal/riichi/calc` pattern from the rules-engine change. Splitting Update / View into separate files is a common bubbletea idiom but not worth doing for this small slice — they live in `play.go` together until they grow.

Alternatives considered:
- **Single file under `cmd/play.go`**: rejected; would re-tangle command parsing with the program and force a refactor in the next change.
- **Public `pkg/play`**: rejected; `internal/` is correct while the API is settling.

### Renderer interface with two implementations, selected once at startup

```go
type Renderer interface {
    Tile(t tile.Tile, focused bool) string  // a single tile, possibly highlighted
    Back() string                            // a face-down tile
}
```

`UnicodeRenderer` and `ASCIIRenderer` implement it. The `--ascii` flag determines which is constructed at program startup; once chosen, the renderer doesn't change for the program's lifetime. This keeps the rendering layer swappable without reconstructing every frame.

The Unicode implementation appends VS-15 (U+FE0E) to each glyph to force text-presentation rather than emoji-color, which keeps cell width consistent across terminals that otherwise default to color emoji rendering. The ASCII implementation uses `┌──┐ │1m│ └──┘` boxed forms — 4 columns wide, 3 rows tall per tile, identical to the layout draft from the discussion.

Alternatives considered:
- **Pick one rendering mode and ship it**: rejected; the user explicitly wants both supported during the prove-it-on-my-terminal phase.
- **Live toggle between modes via key**: rejected; over-engineering for a skeleton that hardcodes everything else.

### Fixed 80×24 layout, dimensions stored but ignored by View()

The model carries `width` and `height` fields populated from `tea.WindowSizeMsg`. View() uses them only to decide between two render paths:

- Both ≥ 80 and ≥ 24 respectively → render the full layout (centered if the actual size is larger)
- Either is smaller → render a single-line "terminal too small (need 80×24)" notice

Dynamic reflow / compact mode / wide-mode sidebar are deferred. Capturing dimensions now means adding reflow later is purely a View() change — the model already has the data.

Alternatives considered:
- **Don't store dimensions until we use them**: rejected; tiny addition now, prevents a model rewrite later.
- **Reflow on day one**: rejected; tripled the layout-code complexity and unnecessary while game state is the next big problem.

### Keybinding map — keyboard only, action keys bound but inert

The full keymap is fixed in this change because muscle memory forms quickly. Action keys are bound and shown in the footer but produce no game-state change because there is no game state.

| Key | Action this change | Future binding |
|---|---|---|
| `←` `→` `h` `l` | Move cursor across hand | Same |
| `1`–`9` | Jump cursor to nth tile | Same |
| `d` / Enter | Highlight discard target (visual only) | Discard the tile (game-loop) |
| `r` | Visual ack | Riichi declaration (game-loop) |
| `t` | Visual ack | Tsumo (game-loop) |
| `p` `c` `k` | Visual ack (greyed in footer) | Pon / Chi / Kan (game-loop, with legality from engine) |
| Space | Visual ack | Pass / no-call (game-loop) |
| `?` | Visual ack | Machi / yaku peek (trainer-aid) |
| `q`, Ctrl+C | Quit | Same |

Mouse support is intentionally absent. It can be added later as a `tea.MouseClickMsg` handler in Update without disturbing the existing keymap.

Alternatives considered:
- **Vim-style `j/k` for vertical movement**: rejected; the hand is horizontal, so `h/l` plus arrows is natural and `j/k` would be misleading.
- **Defer the keymap to game-loop**: rejected; the discussion settled it explicitly so muscle memory builds correctly from the start.

### Hardcoded fixture is the chinitsu+toitoi+sanankou smoke-test hand

The player's hand is hardcoded to `1m1m1m4m4m4m7m7m7m9m9m9m5m5m` — a known winning hand from the existing golden test. Opponents are rendered as 13 face-down tiles each. Sample discards in the centre pond are fixed dummy values (a few yaochuhai per side). Status line values (round/honba/wall/scores) are hardcoded constants.

This means every visual element is observable on the screen during smoke testing — empty discards or empty hands wouldn't tell us if rendering works.

Alternatives considered:
- **Empty hand placeholder**: rejected; doesn't exercise tile rendering.
- **Random hand each launch**: rejected; non-determinism makes "does it look right?" harder to evaluate.

### Test strategy: manual smoke-test only

No automated tests are written in this change. Verification is the author launching `mahjong play` and `mahjong play --ascii` in their target terminal and confirming the layout, cursor movement, and quit behavior. Golden-frame tests (capturing View() output to a string and comparing) come back in `add-game-loop` once there is real state worth pinning.

Alternatives considered:
- **Golden-frame tests now**: rejected; tests against a hardcoded layout become churn the moment we add state. Better to add them when the surface stabilizes.

## Risks / Trade-offs

- **Unicode glyph cell-width drift in unexpected terminals.** → Mitigation: VS-15 monochrome forcing; if it still drifts, `--ascii` is the escape hatch and can be made the default later.
- **Bubbletea v2 API is still maturing.** → Mitigation: pin to a specific minor version in go.mod and revisit when v2 stabilizes; the model/Update/View split is API-stable enough that minor-version bumps won't cascade.
- **No tests means visual regressions are easy to introduce.** → Accepted; the next change introduces golden-frame tests once there's stable state to pin. The skeleton is small enough that a single manual smoke-test catches most issues.
- **Hardcoded fixture will be tedious to update if rendering changes.** → Accepted; the fixture is one short string in `internal/play/play.go` and replacing it with engine-driven values is the first task of `add-game-loop`.
- **80×24 minimum will exclude some users (e.g., narrow phone-tethered terminals).** → Accepted for v1; the dimensions are stored on the model so a future change can add a compact mode without restructuring.
