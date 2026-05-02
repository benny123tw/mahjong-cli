# play-screen Specification

## Purpose

TBD - created by archiving change 'add-tui-skeleton'. Update Purpose after archive.

## Requirements

### Requirement: Play Subcommand Launch

The system SHALL expose the play screen via the `mahjong play` cobra subcommand with three flags: `--ascii` (boolean, default false), `--seed <integer>`, and `--no-akadora` (boolean, default false). The subcommand SHALL launch a bubbletea v2 program that constructs a `*game.Match` (which builds the per-hand `*game.Game`) from the seed (or an OS-derived random seed when omitted) and renders the live game state. The subcommand SHALL exit cleanly when the user presses `q` or sends Ctrl+C.

When `--no-akadora` is omitted or set to false (the default), the subcommand SHALL construct the match via `game.NewMatch(seed)` so akadora is enabled. When `--no-akadora` is set to true, the subcommand SHALL construct the match via `game.NewMatchWithOptions(seed, game.MatchOptions{Akadora: false})` so the wall contains zero red tiles for the entire hanchan.

#### Scenario: Default launch starts a new randomly-seeded game

- **WHEN** the user runs `mahjong play`
- **THEN** the program prints a `Seed: <integer>` line at startup and starts an interactive game with bot opponents in seats East, West, North (the human is South by default)
- **AND** the wall contains one red copy of each five (akadora is on by default)

#### Scenario: ASCII flag selects the ASCII renderer

- **WHEN** the user runs `mahjong play --ascii`
- **THEN** the play screen renders using the ASCII boxed renderer for the player's hand and the ASCII compact renderer for ponds
- **AND** no Unicode mahjong glyphs (U+1F000 block) appear anywhere in the rendered output

#### Scenario: Seed flag pins the game to a deterministic sequence

- **WHEN** the user runs `mahjong play --seed 42`
- **THEN** the wall, dealing, and all bot probabilistic decisions are derived from seed 42
- **AND** running `mahjong play --seed 42` again produces a byte-identical sequence of game events

#### Scenario: No-akadora flag disables red fives for the whole match

- **WHEN** the user runs `mahjong play --no-akadora --seed 42`
- **THEN** every wall constructed during the hanchan (all 8+ hands) contains zero red tiles
- **AND** the dora/ura-dora/akadora-han contribution from red fives is zero throughout the match
- **AND** running `mahjong play --no-akadora --seed 42` again produces a byte-identical sequence of game events

#### Scenario: Ctrl+C exits cleanly

- **WHEN** the user sends Ctrl+C while the program is running
- **THEN** the program returns from its main loop with no error and the terminal state is restored


<!-- @trace
source: add-akadora
updated: 2026-05-02
code:
  - internal/game/wall.go
  - internal/game/state.go
  - testdata/game/golden/seed-42.json
  - internal/game/kan.go
  - internal/game/call.go
  - internal/game/payout.go
  - internal/play/kan_keys.go
  - internal/play/play.go
  - cmd/play.go
  - internal/game/bot.go
  - internal/game/match.go
  - internal/game/turn.go
tests:
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/kan_test.go
  - internal/game/bot_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
-->

---
### Requirement: Play Screen Layout

The system SHALL render a play layout at fixed 80 columns by 24 rows containing the following regions in documented fixed positions: a status line at the top; a toimen (opposite seat) horizontal tile-back row plus seat label; **four per-seat discard zones** — one for each seat (toimen above, your zone below, kamicha on the left, shimocha on the right) — each rendering up to 12 most-recent discards in 6-wide sub-rows, with older discards scrolling off the top with a `+N earlier` indicator; a centre region showing round wind, honba count, wall-remaining count, and the active dora indicator tile; the player's hand at the bottom, rendered as a sorted 13-tile main row with the just-drawn 14th tile visually separated at the right end by a single tile-slot gap when the state is `AwaitingDiscard{Human}`; and an action button row footer that doubles as the call-window prompt when applicable.

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

#### Scenario: Drawn tile is visually separated from the sorted main hand

- **GIVEN** the underlying state is `AwaitingDiscard{Human}` and the human's hand has 14 tiles (13 sorted + the just-drawn tile at index 13)
- **WHEN** the View renders the hand region
- **THEN** the leftmost 13 tiles render densely as one block in canonical sort order
- **AND** a one-tile-slot horizontal gap appears between the 13th rendered tile and the 14th tile
- **AND** the 14th tile renders alone at the rightmost slot

##### Example: gap layout in Unicode mode

- **GIVEN** sorted hand `[1m, 1m, 2m, 3m, 5p, 5p, 6p, 7p, 1s, 1s, 7z, 7z, 7z]` and drawn tile `4m`
- **WHEN** the View renders the hand region
- **THEN** the rendered string is the 13 sorted tiles glued together with no separator (each tile being `<glyph><VS-15> ` per the Tile Rendering Strategy), then a single empty tile-slot of horizontal whitespace, then the rendered drawn tile `4m`

#### Scenario: No gap when not in AwaitingDiscard

- **GIVEN** the underlying state is `AwaitingClaims`, `AwaitingDraw`, or `RoundOver`, with the human's hand at exactly 13 tiles
- **WHEN** the View renders the hand region
- **THEN** all 13 tiles render densely with no gap; the drawn-tile separator only appears in `AwaitingDiscard{Human}`


<!-- @trace
source: add-hand-sort
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/state.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
tests:
  - internal/play/play_test.go
  - internal/game/sort_test.go
-->

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

---
### Requirement: Keybinding Map

The system SHALL bind the documented keymap. Cursor-movement keys SHALL update the focused tile within the player's hand. Action keys SHALL drive real game-state transitions: `D` or Enter discards the focused tile; `R` declares riichi (when legal — concealed hand at tenpai with at least 1000 points); `T` declares tsumo on a winning drawn tile; `P` / `C` / `K` / `R` / `Space` operate inside the call-window prompt. The `K` key in `StateAwaitingDiscard{Player: HumanSeat}` SHALL open the kan picker (ankan or shouminkan) when the human has any eligible kan declaration; in `StateAwaitingClaims` it SHALL submit a `ClaimKan` minkan claim when the human's hand contains 3 of the discarded tile. The `K` key SHALL be greyed when the human has declared riichi (riichi-and-kan interaction deferred).

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

#### Scenario: Kan key opens picker when ankan is eligible

- **GIVEN** the underlying state is `AwaitingDiscard{Player: HumanSeat}` and the human's concealed hand contains four `5p` tiles
- **WHEN** the human presses `K`
- **THEN** the footer shows a kan picker listing `5p` as the eligible declaration
- **AND** pressing the corresponding selector key submits `InputDeclareAnkan{TileID: tile.P5}`

#### Scenario: Kan key submits minkan in claim window

- **GIVEN** the underlying state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}` and the human has three `5p` in hand
- **WHEN** the human presses `K`
- **THEN** the engine receives `Claim{Kind: ClaimKan}` for the human seat and the round transitions to `StateAwaitingDiscard{Player: HumanSeat}` after the kan resolves

#### Scenario: Kan key greyed during human's riichi

- **GIVEN** the human has declared riichi (`Game.IsRiichiDeclared(HumanSeat)` is true)
- **WHEN** the human presses `K` in `StateAwaitingDiscard`
- **THEN** the kan declaration does not fire and the footer shows a greyed indicator

##### Example: full keymap

| Key            | Behavior in this change                                                                  |
| -------------- | ---------------------------------------------------------------------------------------- |
| `←`, `→`, h, l | Move cursor across the player's hand                                                     |
| 1–9            | Jump cursor to nth tile (also picks kan option when picker is open)                      |
| D, Enter       | Discard tile under cursor (when in `AwaitingDiscard` state)                              |
| R              | Declare riichi (when legal: concealed, tenpai, ≥ 1000 points)                            |
| T              | Tsumo on the drawn tile (when winning hand with at least one yaku, including rinshan)    |
| P              | Pon (in call window only, when legal)                                                    |
| C              | Chi (in call window only, only from kamicha, when legal)                                 |
| K              | Open ankan/shouminkan picker (own turn) or submit minkan claim (call window); greyed when in riichi |
| R              | Ron (in call window OR chankan window when winning hand with at least one yaku)          |
| Space          | Pass in call/chankan window; no-op outside call windows                                  |
| ?              | Machi/yaku peek (cached `hand.Shanten` + `hand.Machi` lookup)                            |
| q, Ctrl+C      | Quit cleanly                                                                             |


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Call Window Prompt

The system SHALL render a call-window prompt in the action footer whenever the underlying game-loop state is `AwaitingClaims` and the human player has at least one legal claim available. Each call key SHALL be live or greyed based on real-time legality checks against the human's hand and game state. The hardcoded `[R]on (greyed)` placeholder SHALL be replaced with state-derived rendering: live `[R]on` when `calc.Analyze` returns non-nil for `concealed + discard` AND `Game.IsFuriten(HumanSeat)` is false; greyed `[R]on (furiten)` when the hand would otherwise win but furiten blocks it; greyed `[R]on` (no suffix) when the hand simply does not win. `[K]an (greyed)` SHALL remain a hardcoded placeholder until kan support lands. Pressing `Space` SHALL submit a pass and SHALL transition the state machine via the no-claim path. Greyed keys SHALL NOT advance state. The prompt SHALL NOT enforce a real-time timeout — the player SHALL be allowed unbounded wall-clock thinking time.

#### Scenario: Call window appears after opponent discard with legal pon

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: West}` and the human player has two 5p (legal pon)
- **WHEN** the View is rendered
- **THEN** the action footer shows `[P]on` rendered live
- **AND** pressing `P` advances state to a discard step for the human player with the new pon meld registered

#### Scenario: Live [R]on when ron is legal

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns non-nil
- **AND** `Game.IsFuriten(HumanSeat) == false`
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on` rendered live with no `(furiten)` suffix
- **AND** pressing `R` advances state to `RoundOver{Outcome: OutcomeRon{...}}`

#### Scenario: [R]on greyed with furiten suffix

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns non-nil
- **AND** `Game.IsFuriten(HumanSeat) == true`
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on (furiten)` rendered greyed
- **AND** pressing `R` does not change state and sets `ackText` to a substring containing "furiten"

#### Scenario: [R]on greyed without suffix when no winning shape

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns nil
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on` rendered greyed with no suffix
- **AND** pressing `R` does not change state

#### Scenario: Pass advances state with no claim

- **GIVEN** a call window is active and the human player has no winning hand and no other legal claim
- **WHEN** the human presses `Space`
- **THEN** the human's pass is recorded and the state machine resolves the claim window with no winner from the human's side

#### Scenario: Greyed keys do not mutate state

- **GIVEN** a call window is active where chi is not legal (discarder is not kamicha)
- **WHEN** the human presses `C`
- **THEN** state does not change and a brief footer feedback indicates the key is illegal


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
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

---
### Requirement: Human Riichi Key Binding

The system SHALL bind the `r` key in `StateAwaitingDiscard{Player: HumanSeat}` to riichi declaration: when pressed, submit `InputDiscard{Index: cursor, Riichi: true}` to the engine. When the engine returns `ErrIllegalRiichi`, the TUI SHALL set `ackText` to a description of which precondition failed (e.g., "riichi: hand not tenpai", "riichi: insufficient funds", "riichi: wall has <4 tiles", "riichi: hand is open"). When the engine accepts the riichi, the TUI SHALL clear `ackText` and rely on the new state (advanced to `StateAwaitingClaims`) to drive subsequent rendering.

#### Scenario: R in discard state declares riichi when legal

- **GIVEN** the human is in `StateAwaitingDiscard{Player: HumanSeat}` with cursor at index 13 and a tenpai post-discard hand
- **WHEN** the user presses `r`
- **THEN** the underlying game state advances to `StateAwaitingClaims{Discarder: HumanSeat}`
- **AND** the TUI's `ackText` is empty

#### Scenario: R rejected with descriptive ackText when illegal

- **GIVEN** the human is in `StateAwaitingDiscard{Player: HumanSeat}` and the post-discard hand at cursor index 5 is NOT tenpai
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains a substring describing the failure (e.g., "tenpai", "riichi")


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Human Ron Key Binding

The system SHALL bind the `r` key in `StateAwaitingClaims{Discarder: !=HumanSeat}` to ron declaration: when pressed AND `calc.Analyze` on `concealed + discard` returns non-nil AND `Game.IsFuriten(HumanSeat)` is false, the TUI SHALL submit `InputResolveClaims{Claims: {HumanSeat: ClaimRon}}` to the engine. When pressed AND ron is illegal, the TUI SHALL set `ackText` to either "ron: no yaku" (when `calc.Analyze` returns nil) or "ron: furiten" (when furiten check fails), and leave game state unchanged.

#### Scenario: R in claim window declares ron when legal

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand wins yakufully on 5p with no furiten
- **WHEN** the user presses `r`
- **THEN** the game advances to `StateRoundOver{Outcome: OutcomeRon{...}}`

#### Scenario: R rejected with "no yaku" ackText

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand completes a winning shape on 5p but has no yaku
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains "no yaku"

#### Scenario: R rejected with "furiten" ackText

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand wins yakufully on 5p
- **AND** the human's own pond contains 5p (permanent furiten)
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains "furiten"

<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Bot Action Dispatch

The system SHALL dispatch bot actions through `handleBotTick` based on game state. The dispatcher SHALL invoke each of the following decisions in order, short-circuiting on the first that produces an action:

For `StateAwaitingDraw{Player: bot}`:
- Submit `InputDraw{}`.

For `StateAwaitingDiscard{Player: bot}`:
1. Tsumo check — build the bot's 14-tile `hand.Hand` with the drawn tile as `Winning` and call `calc.Analyze`. If non-nil, submit `InputDeclareTsumo` and stop.
2. Riichi check — invoke `Bot.ShouldRiichi(hand, scores[bot], wall.LiveRemaining(), isHandOpen)`. If `(declare=true, idx=<n>)`, submit `InputDiscard{Index: n, Riichi: true}` and stop.
3. Fallback — submit `InputDiscard{Index: bot.PickDiscard(hand)}`.

For `StateAwaitingClaims{Discarder: anySeat}`:
1. For each non-discarder seat in seat order East, South, West, North (skipping the discarder), evaluate in priority order: (a) ron — `calc.Analyze(concealed+discard) != nil` AND `!Game.IsFuriten(seat)` → `Claim{Kind: ClaimRon}`; (b) pon — `Bot.ShouldPon(hand, discard, isYakuhai, shanten)` → `Claim{Kind: ClaimPon}`; (c) chi (kamicha only) — `Bot.ShouldChi(hand, discard, discarder, seat)` returns a non-empty option list → `Claim{Kind: ClaimChi, ChiTiles: options[0]}`; (d) otherwise no claim.
2. Submit `InputResolveClaims{Claims: collectedMap}` in a single call. The engine's `ResolveClaims` enforces ron > pon > chi priority and the head-bump tiebreak.

When the human is also a non-discarder in claims state, the bot dispatcher SHALL NOT submit on the human's behalf — the human's own keypress drives their `Claim` (or pass via Space).

#### Scenario: Bot tsumo dispatched on winning draw

- **GIVEN** the active state is `AwaitingDiscard{Player: SeatEast}` and East's 14-tile hand wins yakufully
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputDeclareTsumo` and transitions to `StateRoundOver{Outcome: OutcomeTsumo{Winner: SeatEast}}`

#### Scenario: Bot ron dispatched in claim window

- **GIVEN** the active state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}`
- **AND** SeatNorth's `concealed + 5p` forms a yaku-bearing winning hand and SeatNorth is not in furiten
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: {SeatNorth: ClaimRon}}` and transitions to `StateRoundOver{Outcome: OutcomeRon{Winner: SeatNorth, Loser: SeatEast}}`

#### Scenario: Bot pon dispatched in claim window

- **GIVEN** the active state is `AwaitingClaims{Discard: East-wind, Discarder: SeatEast}`
- **AND** SeatNorth has two East-wind tiles (yakuhai pon legal)
- **AND** SeatNorth is not at a winning shape on the discard
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: {SeatNorth: ClaimPon}}` and transitions to `StateAwaitingDiscard{Player: SeatNorth}`

#### Scenario: Bot riichi dispatched on tenpai discard turn

- **GIVEN** the active state is `AwaitingDiscard{Player: SeatWest}` and West's 14-tile hand has a discardable index leaving a tenpai 13-tile shape, score ≥1000, wall ≥4, hand concealed
- **AND** West's hand is NOT in a winning shape (no tsumo)
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputDiscard{Index: <first-tenpai-leaving-index>, Riichi: true}`

#### Scenario: Multiple bots claim the same discard — resolver picks the winner

- **GIVEN** the active state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}`
- **AND** both SeatSouth (the human, hypothetically) and SeatWest can ron on 5p
- **WHEN** the bot dispatcher submits `InputResolveClaims` with both ron claims (in a hypothetical all-bot configuration)
- **THEN** the existing `ResolveClaims` head-bump rule selects the seat closest to the discarder going right around the table (SeatSouth in this example)

#### Scenario: Bot pass when no decision triggers

- **GIVEN** the active state is `AwaitingClaims` and no non-discarder seat has a legal claim
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: nil}` and the round advances to the next player's `AwaitingDraw`

<!-- @trace
source: add-smart-ai
updated: 2026-05-02
code:
  - internal/game/bot.go
  - internal/play/play.go
  - internal/game/turn.go
tests:
  - internal/game/bot_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Match-Bound Model

The play-screen `Model` SHALL hold a `*game.Match` rather than a `*game.Game` so multi-hand state — scores, dealer, round, hand index, honba, riichi sticks — flows through to rendering and transitions. The `NewWithMatch(renderer Renderer, m *game.Match) Model` constructor SHALL be the canonical entry point for the `mahjong play` CLI; `NewWithGame(renderer, g)` MAY remain for tests that need to drive a single round directly. `Model.GameState()` SHALL delegate to `m.match.CurrentGame().State()` so existing per-state rendering paths continue to work unmodified.

#### Scenario: Status bar reflects live match state

- **GIVEN** a model bound to a `*game.Match` at East 2 (handIndex = 1), honba = 1, riichi sticks = 1, scores = `[24000, 25500, 25500, 25000]`
- **WHEN** the model renders the status row
- **THEN** the row displays the round name "East 2", honba 1, riichi pool 1, and each seat's current score (the hardcoded "East 1 · Honba 0 · Score 25000" string SHALL NOT appear)


<!-- @trace
source: add-multi-round
updated: 2026-05-02
code:
  - testdata/game/golden/seed-42.json
  - cmd/play.go
  - internal/play/play.go
  - internal/game/match.go
  - internal/game/bot.go
  - internal/game/turn.go
  - internal/game/payout.go
tests:
  - internal/game/bot_test.go
  - internal/game/match_test.go
  - internal/game/turn_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
-->

---
### Requirement: End-of-Hand Acknowledgement

When the active hand reaches `StateRoundOver`, the model SHALL invoke `Match.AdvanceFromOutcome` once, capture the returned `TransitionResult`, and store it as a pending acknowledgement. While an acknowledgement is pending, the model SHALL render an end-of-hand summary panel showing the outcome (ron/tsumo/ryuukyoku), per-seat point deltas, post-payout totals, and (if renchan) the new honba count. The next hand's `*Game` SHALL NOT begin processing inputs until the player presses any key, at which point the model clears the pending acknowledgement and resumes normal play with the new hand's state machine. If the `TransitionResult` includes a populated `MatchOutcome`, the acknowledgement SHALL transition to the end-of-match standings screen instead of the next hand.

#### Scenario: Ron at East 1 shows ack panel and waits for keypress

- **GIVEN** a model bound to a match at East 1, dealer = `SeatEast`, with the active game at `StateRoundOver{Outcome: OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: ...}}`
- **WHEN** the model receives any tea.Msg
- **THEN** `Match.AdvanceFromOutcome` is invoked exactly once
- **AND** the model renders a panel showing "South ron from East" plus the per-seat deltas
- **AND** the panel remains visible until any key is pressed

#### Scenario: Keypress on ack panel advances to the next hand

- **GIVEN** the model is showing the end-of-hand ack panel for a non-renchan ron
- **WHEN** the player presses any key
- **THEN** the ack panel clears
- **AND** the model's underlying `*Game` is the new East 2 hand (dealer rotated, handIndex incremented)
- **AND** rendering resumes the normal play layout


<!-- @trace
source: add-multi-round
updated: 2026-05-02
code:
  - testdata/game/golden/seed-42.json
  - cmd/play.go
  - internal/play/play.go
  - internal/game/match.go
  - internal/game/bot.go
  - internal/game/turn.go
  - internal/game/payout.go
tests:
  - internal/game/bot_test.go
  - internal/game/match_test.go
  - internal/game/turn_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
-->

---
### Requirement: End-of-Match Standings Screen

When `Match.IsFinished()` returns true, the model SHALL render a standings screen listing all four seats sorted by final score descending, plus the match-end reason ("hanchan-complete" or "tobi: <seat>"). The screen SHALL accept the `q` key to quit and SHALL ignore all other inputs (no further hands to play). The standings screen SHALL replace the normal play layout entirely (no status bar, no pond zones, no hand row).

#### Scenario: Standings shown on hanchan completion

- **GIVEN** a match that has just finished South 4 with non-renchan outcome and `Match.FinalOutcome().Reason == "hanchan-complete"`, scores `[26500, 24500, 27500, 21500]`
- **WHEN** the model renders
- **THEN** the layout shows four rows in descending order: West 27500, East 26500, South 24500, North 21500
- **AND** the reason "hanchan-complete" is displayed
- **AND** pressing `q` quits the program

#### Scenario: Standings shown on tobi

- **GIVEN** a match that ended due to tobi with `Match.FinalOutcome().Reason == "tobi"`, `BustSeat = SeatNorth`
- **WHEN** the model renders
- **THEN** the standings display the four seats with their final scores
- **AND** the reason "tobi: North" (or equivalent) is shown

<!-- @trace
source: add-multi-round
updated: 2026-05-02
code:
  - testdata/game/golden/seed-42.json
  - cmd/play.go
  - internal/play/play.go
  - internal/game/match.go
  - internal/game/bot.go
  - internal/game/turn.go
  - internal/game/payout.go
tests:
  - internal/game/bot_test.go
  - internal/game/match_test.go
  - internal/game/turn_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
-->