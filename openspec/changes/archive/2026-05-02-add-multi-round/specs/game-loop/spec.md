## ADDED Requirements

### Requirement: Per-Hand Dealer-Relative Seat Wind

The system SHALL compute each seat's wind dealer-relative on a per-hand basis rather than from a fixed `Seat → wind` mapping. The system SHALL expose `Game.SeatWindFor(seat Seat) uint8` returning `tile.EastWind + uint8((seat - dealer + 4) % 4)`. Engine code paths that populate `calc.Context.SeatWind` SHALL call `Game.SeatWindFor(winner)` rather than the deprecated `Seat.SeatWind()` method. The legacy `Seat.SeatWind()` MUST continue to exist for the standalone `mahjong calc` CLI, where the user supplies seat winds directly. The `Game` constructor variant `NewWithDealer(seed int64, dealer Seat, roundWind uint8) *Game` SHALL accept the dealer seat and round wind explicitly; `New(seed int64) *Game` SHALL delegate to `NewWithDealer(seed, SeatEast, tile.EastWind)` for backwards compatibility.

#### Scenario: East-1 hand pins seat winds to seat IDs

- **GIVEN** `g := game.NewWithDealer(7, SeatEast, tile.EastWind)`
- **WHEN** the caller queries `g.SeatWindFor(SeatEast)`, `g.SeatWindFor(SeatSouth)`, `g.SeatWindFor(SeatWest)`, `g.SeatWindFor(SeatNorth)`
- **THEN** the returned values are `EastWind`, `SouthWind`, `WestWind`, `NorthWind` (matching the legacy `Seat.SeatWind()` exactly)

#### Scenario: East-2 hand rotates seat winds dealer-relative

- **GIVEN** `g := game.NewWithDealer(7, SeatSouth, tile.EastWind)` (East-2 with dealer rotated to SeatSouth)
- **WHEN** the caller queries `g.SeatWindFor(SeatSouth)`, `g.SeatWindFor(SeatWest)`, `g.SeatWindFor(SeatNorth)`, `g.SeatWindFor(SeatEast)`
- **THEN** the returned values are `EastWind`, `SouthWind`, `WestWind`, `NorthWind` (the dealer is always East-wind regardless of physical seat)

#### Scenario: contextForWin reads seat wind via SeatWindFor

- **GIVEN** a game at East-2 (dealer = `SeatSouth`) where `SeatNorth` (now West-wind for this hand) wins by tsumo
- **WHEN** `Game.contextForWin(SeatNorth, true)` is invoked
- **THEN** the returned `calc.Context.SeatWind` is `tile.WestWind` (not `tile.NorthWind`)
