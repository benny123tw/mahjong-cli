## MODIFIED Requirements

### Requirement: Hanchan Match Structure

The system SHALL model a hanchan match as exactly 8 hands indexed 0..7: indices 0-3 are the East round (round wind = `tile.EastWind`), and indices 4-7 are the South round (round wind = `tile.SouthWind`). The match SHALL be created via `match.NewMatch(seed int64) *Match` initializing all four seats' scores to 25000, dealer = `SeatEast`, hand index = 0, honba = 0, riichi sticks = 0, round wind = `tile.EastWind`, and the active per-hand `*Game` constructed via `game.NewWithDealer(seed, SeatEast, tile.EastWind)`. `NewMatch` SHALL be equivalent to `NewMatchWithOptions(seed, MatchOptions{Akadora: true})` so that the default play experience includes akadora.

The system SHALL also expose `match.NewMatchWithOptions(seed int64, opts MatchOptions) *Match` accepting `MatchOptions{Akadora bool}`. The match SHALL store the options and thread them through to every per-hand `*Game` constructed for indices 0..7 (both at match start and on every dealer rotation / renchan / honba advance), so the akadora setting applies uniformly across all 8 hands of the match. Per-hand `*Game` construction SHALL use a constructor that forwards the akadora setting through to the wall (e.g. `game.NewWithDealerOptions(seed, dealer, roundWind, GameOptions{Akadora: opts.Akadora})`).

#### Scenario: Fresh match starts at East 1 with all seats at 25000

- **GIVEN** `match.NewMatch(7)` is called
- **WHEN** the caller queries `Match.Scores()`, `Match.Dealer()`, `Match.HandIndex()`, `Match.Honba()`, `Match.RoundWind()`
- **THEN** scores are `[25000, 25000, 25000, 25000]`, dealer is `SeatEast`, hand index is 0, honba is 0, round wind is `tile.EastWind`

#### Scenario: Default NewMatch enables akadora across all hands

- **GIVEN** `match.NewMatch(7)` is called
- **WHEN** the active per-hand `*Game`'s wall is inspected
- **THEN** the wall contains exactly one red copy of each five (5m, 5p, 5s)
- **AND** after `AdvanceFromOutcome` rotates to the next hand, the new hand's wall ALSO contains exactly one red copy of each five

#### Scenario: NewMatchWithOptions threads akadora-off to every hand

- **GIVEN** `match.NewMatchWithOptions(7, MatchOptions{Akadora: false})` is called
- **WHEN** the active hand's wall is inspected, then the match is advanced through several hands
- **THEN** every constructed wall contains zero red tiles (no `Red == true`)
- **AND** each wall still contains 4 copies of every tile ID
