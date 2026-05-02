## ADDED Requirements

### Requirement: Human Hand Canonical Sort Invariant

The system SHALL maintain the human player's concealed hand in canonical sort order whenever it has 13 tiles. Canonical order is ascending tile ID: M1, M2, ..., M9, P1, P2, ..., P9, S1, S2, ..., S9, EastWind, SouthWind, WestWind, NorthWind, Haku, Hatsu, Chun. Sorting SHALL be triggered after every mutation of the human's concealed hand: initial deal, after the human discards, after the human's hand is altered by a successful call (pon / chi). Bot seats' hands are NOT sorted — bot decision logic is order-independent and a sort would be wasted work.

When the human's state is `AwaitingDiscard{Player: Human}`, their hand SHALL contain exactly 14 tiles where the leftmost 13 are in canonical sort order and the 14th tile (the just-drawn tile) is appended at index 13 WITHOUT being merged into the sort. The 14th tile SHALL retain its drawn-tile position regardless of where it would fall in canonical order, so the player can identify which tile they just drew.

After the human discards (either the drawn 14th tile or any of the sorted 0..12 tiles), the resulting 13-tile hand SHALL be re-sorted into canonical order before the next state transition completes.

#### Scenario: Initial deal sorts the human's hand

- **GIVEN** a new game starts with `--seed 42` and the human is seated South
- **WHEN** the wall is dealt
- **THEN** the human's 13-tile hand at index 0..12 is in canonical ascending tile ID order
- **AND** no two adjacent tiles violate `hand[i].ID <= hand[i+1].ID`

#### Scenario: Drawn tile lives at index 13 unsorted

- **GIVEN** the human's sorted 13-tile hand contains tiles ending at `S5` (ID 22)
- **WHEN** the human draws a tile with ID `M3` (ID 2, which would canonically sort to position 2)
- **THEN** state becomes `AwaitingDiscard{Player: Human}`
- **AND** `Game.Hand(Human)` returns 14 tiles where index 13 is the drawn `M3`
- **AND** indices 0..12 remain the previously sorted 13 tiles

#### Scenario: Discarding the drawn tile leaves a sorted 13-tile hand

- **GIVEN** the human's hand is `[sorted 13 tiles, drawn M3]` (14 tiles)
- **WHEN** the human discards index 13 (the drawn `M3`)
- **THEN** the next state has the human's 13-tile hand still in canonical sort order

#### Scenario: Discarding a sorted-hand tile re-sorts after the drawn tile slots in

- **GIVEN** the human's hand is `[1m, 1m, 2m, 3m, 5p, 5p, 6p, 7p, 1s, 1s, 7z, 7z, 7z, drawn=4m]`
- **WHEN** the human discards `1s` (the tile at sorted-hand index 8)
- **THEN** the resulting 13-tile hand is `[1m, 1m, 2m, 3m, 4m, 5p, 5p, 6p, 7p, 1s, 7z, 7z, 7z]` in canonical sort order

#### Scenario: After-call hand is re-sorted

- **GIVEN** the human's sorted 13-tile hand contains two `5p` and an opponent discards `5p`
- **WHEN** the human calls pon and selects a discard
- **THEN** the human's resulting concealed-hand portion (13 tiles minus the 3 melded into the open pon) is in canonical sort order
- **AND** the called meld is recorded separately and does not participate in the concealed-hand sort

#### Scenario: Bot hands are not sorted

- **GIVEN** any bot seat receives a 13-tile deal
- **WHEN** the bot's `Game.Hand(seat)` view is read at any point
- **THEN** there is no ordering guarantee on the bot's tiles
- **AND** the engine SHALL NOT spend cycles maintaining a sort for bot seats
