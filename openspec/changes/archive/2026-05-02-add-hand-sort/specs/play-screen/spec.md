## MODIFIED Requirements

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

