## ADDED Requirements

### Requirement: Call Window Prompt

The system SHALL render a call-window prompt in the action footer whenever the underlying game-loop state is `AwaitingClaims` and the human player has at least one legal claim available. Only legal-call keys SHALL be live; illegal calls (e.g., chi when the discarder is not the player's kamicha, ron without a winning hand) SHALL be rendered greyed and SHALL NOT advance state. Pressing `Space` SHALL submit a pass and SHALL transition the state machine via the no-claim path. The prompt SHALL NOT enforce a real-time timeout — the player SHALL be allowed unbounded wall-clock thinking time.

#### Scenario: Call window appears after opponent discard with legal call

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: West}` and the human player has two 5p (legal pon)
- **WHEN** the View is rendered
- **THEN** the action footer shows `[P]on  [C]hi (greyed)  [K]an (greyed)  [R]on (greyed)  [Space] Pass`
- **AND** pressing `P` advances state to a discard step for the human player with the new pon meld registered

#### Scenario: Pass advances state with no claim

- **GIVEN** a call window is active and the human player has no winning hand
- **WHEN** the human presses `Space`
- **THEN** the human's pass is recorded and the state machine resolves the claim window with no winner from the human's side

#### Scenario: Greyed keys do not mutate state

- **GIVEN** a call window is active where chi is not legal (discarder is not kamicha)
- **WHEN** the human presses `C`
- **THEN** state does not change and a brief footer feedback indicates the key is illegal

### Requirement: Engine Wiring For Game State

The system SHALL invoke `calc.Analyze` from `internal/riichi/calc` whenever the human player attempts tsumo or ron, passing a fully-populated context that includes the seat wind, round wind, riichi state, every revealed dora indicator, and the eight Group C state flags (`Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `DoubleRiichi`, `Tenhou`, `Chiihou`) from the game-loop state. The system SHALL also invoke `hand.Shanten` and `hand.Machi` on demand when the player presses `?` to inspect their wait state — these results SHALL be cached on the model until the hand changes.

#### Scenario: Tsumo declaration triggers full engine analysis

- **GIVEN** the human player draws a tile that completes their hand with riichi declared and ippatsu still alive
- **WHEN** the player presses `T` (tsumo)
- **THEN** the system calls `calc.Analyze` with `Context{Riichi: true, Ippatsu: true, ..., dora indicators}`
- **AND** the result populates a "Win!" overlay with yaku list, han, fu, and points

#### Scenario: Yakuless win is rejected at the TUI surface

- **GIVEN** the human's hand reaches a winning shape with no yaku
- **WHEN** the player presses `T`
- **THEN** `calc.Analyze` returns nil and the TUI shows a brief "no yaku — cannot win" footer message
- **AND** state does not advance to RoundOver

## MODIFIED Requirements

### Requirement: Play Subcommand Launch

The system SHALL expose the play screen via the `mahjong play` cobra subcommand with two flags: `--ascii` (boolean) and `--seed <integer>`. The subcommand SHALL launch a bubbletea v2 program that constructs a `*game.Game` from the seed (or an OS-derived random seed when omitted) and renders the live game state. The subcommand SHALL exit cleanly when the user presses `q` or sends Ctrl+C.

#### Scenario: Default launch starts a new randomly-seeded game

- **WHEN** the user runs `mahjong play`
- **THEN** the program prints a `Seed: <integer>` line at startup and starts an interactive game with bot opponents in seats East, West, North (the human is South by default)

#### Scenario: ASCII flag selects the ASCII renderer

- **WHEN** the user runs `mahjong play --ascii`
- **THEN** the play screen renders using the ASCII boxed renderer for the player's hand and the ASCII compact renderer for ponds
- **AND** no Unicode mahjong glyphs (U+1F000 block) appear anywhere in the rendered output

#### Scenario: Seed flag pins the game to a deterministic sequence

- **WHEN** the user runs `mahjong play --seed 42`
- **THEN** the wall, dealing, and all bot probabilistic decisions are derived from seed 42
- **AND** running `mahjong play --seed 42` again produces a byte-identical sequence of game events

#### Scenario: Ctrl+C exits cleanly

- **WHEN** the user sends Ctrl+C while the program is running
- **THEN** the program returns from its main loop with no error and the terminal state is restored

### Requirement: Play Screen Layout

The system SHALL render a play layout at fixed 80 columns by 24 rows containing the following regions in documented fixed positions: a status line at the top; a toimen (opposite seat) horizontal tile-back row plus seat label; **four per-seat discard zones** — one for each seat (toimen above, your zone below, kamicha on the left, shimocha on the right) — each rendering up to 12 most-recent discards in 6-wide sub-rows, with older discards scrolling off the top with a `+N earlier` indicator; a centre region showing round wind, honba count, wall-remaining count, and the active dora indicator tile; the player's hand at the bottom with a cursor highlight; and an action button row footer that doubles as the call-window prompt when applicable.

#### Scenario: All regions render at sufficient terminal size

- **WHEN** the play screen is active and `tea.WindowSizeMsg` reports at least 80 columns and 24 rows
- **THEN** the rendered output contains the status line, toimen tile-backs, four per-seat discard zones, centre round/honba/wall/dora region, player's hand with cursor, and action footer in their documented fixed positions
- **AND** no centre pond is rendered (centre region is reserved for round info, not discards)

#### Scenario: Larger terminal centers the layout

- **WHEN** `tea.WindowSizeMsg` reports a size larger than 80 columns or 24 rows
- **THEN** the 80×24 layout renders centered within the available area
- **AND** no region of the layout reflows or stretches to fill the additional space

#### Scenario: Smaller terminal shows a notice

- **WHEN** `tea.WindowSizeMsg` reports fewer than 80 columns or fewer than 24 rows
- **THEN** the screen renders only a "terminal too small (need 80×24)" notice in place of the play layout

#### Scenario: Discard pond growth wraps every 6 tiles

- **GIVEN** the human player has discarded 7 tiles
- **WHEN** the View renders the player's discard zone
- **THEN** the zone shows 6 tiles in the first sub-row and 1 tile in the second sub-row

#### Scenario: Pond cap at 12 visible discards with overflow indicator

- **GIVEN** the toimen player has discarded 14 tiles in the round so far
- **WHEN** the View renders the toimen discard zone
- **THEN** the zone shows the 12 most recent discards (2 sub-rows of 6)
- **AND** a `+2 earlier` indicator marks that older discards exist

### Requirement: Tile Rendering Strategy

The system SHALL provide three tile-rendering implementations behind a shared `Renderer` interface: the **Unicode renderer** producing U+1F000-block glyphs with U+FE0E (VS-15) appended for monochrome presentation; the **ASCII boxed renderer** producing 4-column × 3-row tiles for the player's hand; and the **ASCII compact renderer** producing 4-column × 1-row `[1m]`-style tiles, used only inside the four discard zones when `--ascii` is active. Without the compact form, four full-boxed pond zones plus toimen tile-backs plus the player's hand exceed the 24-row budget.

#### Scenario: Unicode renderer appends VS-15

- **WHEN** the Unicode renderer produces a tile string
- **THEN** the string contains a U+1F000-block glyph immediately followed by U+FE0E

#### Scenario: ASCII boxed renderer is 4×3 per tile

- **WHEN** the ASCII boxed renderer produces a tile string for the player's hand
- **THEN** the rendered tile occupies exactly 4 columns and 3 rows

#### Scenario: ASCII compact renderer is 4×1 per tile

- **WHEN** the ASCII compact renderer produces a tile string inside a discard zone
- **THEN** the rendered tile occupies exactly 4 columns and 1 row in `[1m]`-style form

#### Scenario: Renderer choice is fixed at startup

- **WHEN** the program is running
- **THEN** the choice of Unicode-vs-ASCII is set once at startup based on `--ascii` and persists until exit

### Requirement: Keybinding Map

The system SHALL bind the documented keymap. Cursor-movement keys SHALL update the focused tile within the player's hand. Action keys SHALL drive real game-state transitions: `D` or Enter discards the focused tile; `R` declares riichi (when legal — concealed hand at tenpai with at least 1000 points); `T` declares tsumo on a winning drawn tile; `P` / `C` / `K` / `R` / `Space` operate inside the call-window prompt. `K` (kan) SHALL render greyed in v1 and SHALL NOT advance state regardless of context.

#### Scenario: Cursor moves right with arrow or l

- **WHEN** the cursor is at hand position i (0-indexed) and the player presses `→` or `l`
- **THEN** the cursor moves to position min(i+1, hand_length-1)

#### Scenario: Cursor moves left with arrow or h

- **WHEN** the cursor is at hand position i and the player presses `←` or `h`
- **THEN** the cursor moves to position max(i-1, 0)

#### Scenario: Number key jumps cursor

- **WHEN** the player presses a key in the range `1`–`9`
- **THEN** the cursor moves to the nth tile (1-indexed) of the hand if n ≤ hand_length, otherwise the cursor moves to the last tile

#### Scenario: Discard advances game state

- **GIVEN** the underlying state is `AwaitingDiscard{Player: human}` and the cursor is on tile T
- **WHEN** the human presses `D` or Enter
- **THEN** tile T is removed from the human's hand and appended to the human's discard zone, and state advances to `AwaitingClaims{Discard: T, Discarder: human}`

#### Scenario: Riichi declaration is rejected when illegal

- **WHEN** the human presses `R` while their hand is open or not yet at tenpai
- **THEN** state does not change and a brief footer feedback indicates riichi is illegal in the current state

#### Scenario: Kan key is greyed in v1

- **WHEN** the human presses `K` at any state
- **THEN** state does not change (kan is not supported in this change)

##### Example: full keymap

| Key            | Behavior in this change                                                  |
| -------------- | ------------------------------------------------------------------------ |
| `←`, `→`, h, l | Move cursor across the player's hand                                     |
| 1–9            | Jump cursor to nth tile                                                  |
| D, Enter       | Discard tile under cursor (when in `AwaitingDiscard` state)             |
| R              | Declare riichi (when legal: concealed, tenpai, ≥ 1000 points)           |
| T              | Tsumo on the drawn tile (when winning hand with at least one yaku)      |
| P              | Pon (in call window only, when legal)                                    |
| C              | Chi (in call window only, only from kamicha, when legal)                |
| K              | Kan (always greyed in v1)                                                |
| R              | Ron (in call window only, when winning hand with at least one yaku)     |
| Space          | Pass in call window; no-op outside call windows                          |
| ?              | Machi/yaku peek (cached `hand.Shanten` + `hand.Machi` lookup)           |
| q, Ctrl+C      | Quit cleanly                                                             |

## REMOVED Requirements

### Requirement: Hardcoded Fixture For Display

**Reason**: The skeleton's hardcoded chinitsu+toitoi+sanankou hand and dummy centre pond are replaced by live game state from `internal/game`. The hand, opponent tile-back counts, and discards are now derived from the running game.

**Migration**: `cmd/play.go` constructs a `*game.Game` from `(seed, opts)` and passes it to `play.New`. The fixture function `fixtureHand` in `internal/play/play.go` is removed; the model's `hand` field is replaced by a `game *game.Game` pointer. Existing manual smoke tests are replaced by golden-game integration tests in `internal/game/golden_test.go` plus continued manual TUI smoke testing.

#### Scenario: Hardcoded fixture is no longer present after this change

- **WHEN** `mahjong play` is launched after this change ships
- **THEN** the rendered hand SHALL come from `*game.Game` and SHALL NOT come from `fixtureHand`
- **AND** searching the codebase SHALL find no `fixtureHand` definition or call site
