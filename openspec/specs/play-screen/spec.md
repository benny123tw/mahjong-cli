# play-screen Specification

## Purpose

TBD - created by archiving change 'add-tui-skeleton'. Update Purpose after archive.

## Requirements

### Requirement: Play Subcommand Launch

The system SHALL expose the play screen via the `mahjong play` cobra subcommand with an optional `--ascii` flag. The subcommand SHALL launch a bubbletea v2 program rendering the play layout from a hardcoded fixture and SHALL exit cleanly when the user presses `q` or sends Ctrl+C.

#### Scenario: Default launch uses the Unicode renderer

- **WHEN** the user runs `mahjong play`
- **THEN** the bubbletea program starts and the play screen renders using the Unicode tile renderer
- **AND** the program exits with status 0 when the user presses `q`

#### Scenario: ASCII flag selects the ASCII renderer

- **WHEN** the user runs `mahjong play --ascii`
- **THEN** the play screen renders using the ASCII boxed renderer
- **AND** no Unicode mahjong glyphs (U+1F000 block) appear anywhere in the rendered output

#### Scenario: Ctrl+C exits cleanly

- **WHEN** the user sends Ctrl+C while the program is running
- **THEN** the program returns from its main loop with no error and the terminal state is restored

---
### Requirement: Play Screen Layout

The system SHALL render a play layout at fixed 80 columns by 24 rows containing the following regions in documented fixed positions: a status line at the top, a toimen (opposite seat) horizontal tile-back row in the upper region, kamicha (left) and shimocha (right) vertical tile-back strips, a centre discard pond, a dora indicator inset, the player's hand at the bottom with a cursor highlight, and an action button row footer.

#### Scenario: All regions render at sufficient terminal size

- **WHEN** the play screen is active and `tea.WindowSizeMsg` reports at least 80 columns and 24 rows
- **THEN** the rendered output contains the status line, toimen tile-backs, kamicha and shimocha tile-backs, centre discard pond, dora indicator, player's hand with cursor, and action footer in their documented fixed positions

#### Scenario: Larger terminal centers the layout

- **WHEN** `tea.WindowSizeMsg` reports a size larger than 80 columns or 24 rows
- **THEN** the 80×24 layout renders centered within the available area
- **AND** no region of the layout reflows or stretches to fill the additional space

#### Scenario: Smaller terminal shows a notice

- **WHEN** `tea.WindowSizeMsg` reports fewer than 80 columns or fewer than 24 rows
- **THEN** the screen renders only a "terminal too small (need 80×24)" notice in place of the play layout

---
### Requirement: Window Size Captured On Model

The system SHALL update the model's width and height fields whenever `tea.WindowSizeMsg` is received, regardless of whether the View method currently consults those values for layout decisions beyond the small/large/centered branching above.

#### Scenario: WindowSizeMsg updates model

- **WHEN** the program receives `tea.WindowSizeMsg{Width: W, Height: H}`
- **THEN** the next call to View observes model.width = W and model.height = H

##### Example: dimension capture

| WindowSizeMsg sent | model.width after | model.height after | View renders |
| ------------------ | ----------------- | ------------------ | ------------ |
| {80, 24}           | 80                | 24                 | Full layout, top-left aligned |
| {120, 40}          | 120               | 40                 | Full layout, centered |
| {60, 24}           | 60                | 24                 | "Terminal too small" notice |
| {80, 20}           | 80                | 20                 | "Terminal too small" notice |

---
### Requirement: Tile Rendering Strategy

The system SHALL provide two interchangeable tile-rendering implementations behind a shared `Renderer` interface. The Unicode renderer SHALL use mahjong glyphs from the U+1F000 block and SHALL append U+FE0E (the text-variation selector) to each glyph to force monochrome presentation. The ASCII renderer SHALL use boxed forms occupying 4 columns by 3 rows per tile.

#### Scenario: Unicode renderer appends VS-15

- **WHEN** the Unicode renderer produces a tile string
- **THEN** the string contains a U+1F000-block glyph immediately followed by U+FE0E

#### Scenario: ASCII renderer is fixed-size 4×3 per tile

- **WHEN** the ASCII renderer produces a tile string
- **THEN** the rendered tile occupies exactly 4 columns and 3 rows in monospace output

#### Scenario: Renderer is fixed for the program lifetime

- **WHEN** the program is running
- **THEN** the renderer selected at startup persists until the program exits
- **AND** no input or message switches the renderer mid-session

##### Example: tile rendering for 1m

| Renderer | Output (single line approximation) |
| -------- | ---------------------------------- |
| Unicode  | 🀇︎ (U+1F007 followed by U+FE0E)   |
| ASCII    | three lines: "┌──┐", "│1m│", "└──┘" |

---
### Requirement: Keybinding Map

The system SHALL bind the documented keymap. Cursor-movement keys SHALL update the focused tile within the player's hand. Action keys SHALL be bound, visible in the action footer, and produce only visual acknowledgement in this change (no game-state mutation, since the change has no game state).

#### Scenario: Cursor moves right with arrow or l

- **WHEN** the cursor is at hand position i (0-indexed) and the player presses `→` or `l`
- **THEN** the cursor moves to position min(i+1, hand_length-1)

#### Scenario: Cursor moves left with arrow or h

- **WHEN** the cursor is at hand position i and the player presses `←` or `h`
- **THEN** the cursor moves to position max(i-1, 0)

#### Scenario: Number key jumps cursor

- **WHEN** the player presses a key in the range `1`–`9`
- **THEN** the cursor moves to the nth tile (1-indexed) of the hand if n ≤ hand_length, otherwise the cursor moves to the last tile

#### Scenario: Action keys produce visual acknowledgement only

- **WHEN** the player presses `d`, Enter, `r`, `t`, `p`, `c`, `k`, Space, or `?`
- **THEN** the model is unchanged except for any visual acknowledgement state (a brief footer flash or equivalent)
- **AND** no game-state field is mutated (because none exists in this change)

##### Example: full keymap

| Key            | Behavior in this change                       |
| -------------- | --------------------------------------------- |
| `←`, `→`, h, l | Move cursor across the player's hand         |
| 1–9            | Jump cursor to nth tile                       |
| d, Enter       | Visual acknowledge (discard intent)           |
| r              | Visual acknowledge (riichi intent)            |
| t              | Visual acknowledge (tsumo intent)             |
| p, c, k        | Visual acknowledge (greyed in footer)         |
| Space          | Visual acknowledge (pass intent)              |
| ?              | Visual acknowledge (machi peek placeholder)   |
| q, Ctrl+C      | Quit cleanly                                  |

---
### Requirement: Hardcoded Fixture For Display

The system SHALL hardcode the player's hand to the tile string `1m1m1m4m4m4m7m7m7m9m9m9m5m5m`, render each opponent's hidden hand as 13 face-down tiles, fill the centre pond with a fixed set of dummy discards, and use fixed constants for status-line values (round, honba, wall count, scores). The system SHALL NOT call rules-engine analysis functions (Decompose, Shanten, Machi, Evaluate, Fu, Compute, Analyze) in this change.

#### Scenario: Hardcoded hand renders

- **WHEN** the play screen is rendered
- **THEN** the player's hand region contains 14 tiles in the order specified by `1m1m1m4m4m4m7m7m7m9m9m9m5m5m`

#### Scenario: Opponents render as backs only

- **WHEN** the play screen is rendered
- **THEN** each of the three opponent regions (toimen, kamicha, shimocha) contains exactly 13 face-down tile renderings and no front-facing tile content
