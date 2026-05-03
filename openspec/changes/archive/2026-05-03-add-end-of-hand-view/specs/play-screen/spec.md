## MODIFIED Requirements

### Requirement: End-of-Hand Acknowledgement

When the active hand reaches `StateRoundOver`, the model SHALL invoke `Match.AdvanceFromOutcome` once, capture the returned `TransitionResult`, and store it as a pending acknowledgement. While an acknowledgement is pending, the model SHALL render an end-of-hand reveal panel that replaces the normal play layout.

The reveal panel SHALL render four seat rows (East, South, West, North), each containing the seat letter, the seat wind, the seat's concealed hand face-up (no `Back()` glyph), and that seat's open melds rendered via the same machinery used for the human's open melds (the `renderOpenMelds` function SHALL be generalized to take a seat parameter; the existing `renderHand` call site SHALL pass `HumanSeat` explicitly so its visible behavior remains byte-identical).

For win outcomes (`OutcomeRon`, `OutcomeTsumo`, including chankan-ron), the reveal panel SHALL:

- Mark the winner's row with a `[W] ` prefix and bold styling.
- Highlight the winning tile within the winner's concealed-hand row using the existing focused-tile style.
- Render a header line identifying the kind (`RON`, `TSUMO`, `CHANKAN RON`), the winner's seat letter, and (for ron) the dealt-in seat letter.
- Render a yaku-list line listing every yaku name with its han value, separated by ` · ` (e.g., `Yaku: Riichi 1 · Pinfu 1 · Tanyao 1`).
- Render a totals line in the form `Han N · Fu M · Base K`.
- Render a deltas line listing the per-seat point delta produced by `ComputePayouts`, in seat order East/South/West/North, with sign (e.g., `East -8000 · South +8000 · West 0 · North 0`).

For ryuukyoku outcomes (`OutcomeRyuukyoku`), the reveal panel SHALL:

- Render a header line `RYUUKYOKU`.
- Append a `tenpai` or `noten` tag inline at the end of each seat's row, based on whether that seat appears in `OutcomeRyuukyoku.TenpaiPlayers`.
- Render a deltas line listing the per-seat point delta produced by `ComputePayouts`, in seat order East/South/West/North, with sign (e.g., `East -1000 · South -1000 · West +3000 · North -1000`).

The reveal panel SHALL render a footer line `[Any key — Continue]` (replacing the action footer keymap while the panel is active).

The next hand's `*Game` SHALL NOT begin processing inputs until the player presses any key, at which point the model clears the pending acknowledgement and resumes normal play with the new hand's state machine. If the `TransitionResult` includes a populated `MatchOutcome`, the acknowledgement SHALL transition to the end-of-match standings screen instead of the next hand.

#### Scenario: Ron at East 1 reveals all four hands and the yaku breakdown

- **GIVEN** a model bound to a match at East 1, dealer = `SeatEast`, with the active game at `StateRoundOver{Outcome: OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: ...}}` and a winning hand whose yaku list is `[{Name: "Riichi", Han: 1}, {Name: "Pinfu", Han: 1}, {Name: "Tanyao", Han: 1}]`, han = 3, fu = 30, award.base = 480, deltas = `[East -3900, South +3900, West 0, North 0]`
- **WHEN** the model receives any tea.Msg
- **THEN** `Match.AdvanceFromOutcome` is invoked exactly once
- **AND** the rendered output contains a `RON` header naming `SeatSouth` as winner and `SeatEast` as discarder
- **AND** the rendered output contains the concealed hand of every seat face-up, plus that seat's open melds
- **AND** the rendered output contains the substring `Yaku: Riichi 1 · Pinfu 1 · Tanyao 1`
- **AND** the rendered output contains the substring `Han 3 · Fu 30 · Base 480`
- **AND** the rendered output contains the substring `East -3900` and `South +3900`
- **AND** the rendered output contains the footer `[Any key — Continue]`
- **AND** the panel remains visible until any key is pressed

#### Scenario: Tsumo reveal includes winner-row highlight and winning-tile highlight

- **GIVEN** the active game is at `StateRoundOver{Outcome: OutcomeTsumo{Winner: SeatSouth, Result: ...}}` with `Result.WinningTile = {ID: P5}` and the winner's concealed hand contains a `5p` tile
- **WHEN** the panel renders
- **THEN** the South row begins with `[W] ` and the South seat label is rendered with bold styling
- **AND** the rendered output contains an ANSI escape sequence applying bold styling to the `5p` glyph within the South row's concealed-hand block

#### Scenario: Chankan ron is identified in the header

- **GIVEN** the active game is at `StateRoundOver{Outcome: OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: ...}}` AND the pending ack records `IsChankan = true`
- **WHEN** the panel renders
- **THEN** the header line contains the substring `CHANKAN RON` instead of `RON`

#### Scenario: Ryuukyoku reveal labels each seat tenpai or noten

- **GIVEN** the active game is at `StateRoundOver{Outcome: OutcomeRyuukyoku{TenpaiPlayers: [SeatSouth, SeatWest]}}`, deltas = `[East -1500, South +1500, West +1500, North -1500]`
- **WHEN** the panel renders
- **THEN** the rendered output contains `RYUUKYOKU` in the header line
- **AND** the East row contains the inline tag `noten`
- **AND** the South row contains the inline tag `tenpai`
- **AND** the West row contains the inline tag `tenpai`
- **AND** the North row contains the inline tag `noten`
- **AND** the deltas line contains `East -1500`, `South +1500`, `West +1500`, `North -1500`

##### Example: ryuukyoku payment cases

| TenpaiPlayers           | East | South | West | North | Notes                                   |
| ----------------------- | ---- | ----- | ---- | ----- | --------------------------------------- |
| `[]` (none)             | 0    | 0     | 0    | 0     | 0/4 case — no transfer                  |
| `[SeatSouth]`           | -1000 | +3000 | -1000 | -1000 | 1/3 case — 3000 from each noten         |
| `[SeatSouth, SeatWest]` | -1500 | +1500 | +1500 | -1500 | 2/2 case — 1500 each direction          |
| `[SeatEast, SeatSouth, SeatWest]` | +1000 | +1000 | +1000 | -3000 | 3/1 case — sole noten pays 3000 |
| `[SeatEast, SeatSouth, SeatWest, SeatNorth]` | 0 | 0 | 0 | 0 | 4/4 case — no transfer |

The deltas line in each case SHALL match the row exactly, in seat order East/South/West/North, signed.

#### Scenario: Keypress on reveal panel advances to the next hand

- **GIVEN** the model is showing the reveal panel for a non-renchan ron
- **WHEN** the player presses any key
- **THEN** the panel clears
- **AND** the model's underlying `*Game` is the new East 2 hand (dealer rotated, handIndex incremented)
- **AND** rendering resumes the normal play layout
