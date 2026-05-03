## MODIFIED Requirements

### Requirement: Play Screen Layout

The system SHALL render a play layout at fixed 80 columns by 24 rows containing the following regions in documented fixed positions: a status line at the top; a toimen (opposite seat) horizontal tile-back row plus seat label; **four per-seat discard zones** — one for each seat (toimen above, your zone below, kamicha on the left, shimocha on the right) — each rendering up to 12 most-recent discards in 6-wide sub-rows, with older discards scrolling off the top with a `+N earlier` indicator; a centre region showing round wind, honba count, wall-remaining count, and the active dora indicator tile; the player's hand at the bottom, rendered as a sorted 13-tile main row with the just-drawn 14th tile visually separated at the right end by a single tile-slot gap when the state is `AwaitingDiscard{Human}`; and an action button row footer that doubles as the call-window prompt when applicable.

For each opponent seat (Kamicha/East, Toimen/North, Shimocha/West), the system SHALL render that seat's open melds inline within the seat's zone, between the seat-label header and the zone's other content (face-down hand row for Toimen, discard pond for Kamicha and Shimocha). The meld block SHALL be produced by `renderOpenMeldsForSeat(seat)` and SHALL use the same seat-source markers (`[E]`/`[S]`/`[W]`/`[N]` attached to the called tile, `[A]` as a meld-level prefix for ankan) as the human's per-hand meld rendering.

When an opponent has zero open melds, the meld region SHALL contribute nothing to the rendered output for that zone — no extra line, no `(none)` label. The rendered output for the zero-meld case SHALL be byte-identical to the pre-change rendering.

When the meld block's rendered width exceeds the zone's column budget (20 columns for Kamicha and Shimocha; 80 columns for Toimen), the system SHALL re-render the meld block as one meld per line stacked vertically via `lipgloss.JoinVertical`, capped at 2 lines. When even the 2-line stack does not fit (the cumulative width of the first M melds across 2 lines exceeds 2 × zoneWidth), the system SHALL render only the first N melds whose cumulative width fits in 2 × zoneWidth, then append a `+K more` suffix on a third line of the zone where K is the count of unrendered melds.

The human seat's per-zone rendering in the four-quadrant layout SHALL NOT render melds in this opponent-style location. The human's melds continue to render to the right of the concealed-hand row per the existing Open Meld Display For Human Player requirement; double-rendering is forbidden.

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

#### Scenario: Opponent zone renders open melds when present

- **GIVEN** Kamicha (East) has called a single pon of 5p from South
- **WHEN** the play screen renders
- **THEN** Kamicha's zone contains the seat label `Kamicha · East · <score>`, then a meld row containing the pon of 5p with the called-tile marker `[S]` attached to the called tile, then the discard pond
- **AND** the meld row appears between the seat label and the discard pond (above the pond)

#### Scenario: Opponent zone with zero melds is unchanged

- **GIVEN** Toimen (North) has zero open melds
- **WHEN** the play screen renders
- **THEN** the Toimen zone rendering is byte-identical to the pre-change output for that zone (no extra line, no placeholder text)

#### Scenario: Wide meld block wraps to a second line

- **GIVEN** Shimocha (West) has called two pons whose combined width exceeds the 20-column zone budget
- **WHEN** the play screen renders
- **THEN** the Shimocha zone renders the two pons stacked vertically (one meld per line) via `lipgloss.JoinVertical`
- **AND** each line's rendered width is at most 20 columns

#### Scenario: Pathological meld count truncates with +K more

- **GIVEN** Kamicha (East) has called 4 ankans
- **WHEN** the play screen renders
- **THEN** the Kamicha zone renders only the first N ankans whose cumulative width fits within 2 × 20 columns
- **AND** a third line in the zone reads `+K more` where K is the count of unrendered melds (4 minus N)

##### Example: zone-width handling cases

| Scenario              | Meld block raw width | Zone width | Rendered shape                        |
| --------------------- | -------------------- | ---------- | ------------------------------------- |
| 1 pon on Kamicha      | ~10 cols             | 20         | one line, fits as-is                  |
| 2 pons on Kamicha     | ~22 cols             | 20         | two lines, one meld each              |
| 3 pons on Kamicha     | ~33 cols             | 20         | two lines fit 2 melds, +1 more        |
| 4 ankans on Kamicha   | ~56 cols             | 20         | two lines fit 2 ankans, +2 more       |
| 3 pons on Toimen      | ~33 cols             | 80         | one line, fits as-is                  |
