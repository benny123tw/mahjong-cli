## MODIFIED Requirements

### Requirement: Wall Construction and Dealing

The system SHALL construct a 136-tile wall (4 copies each of 34 tile types) for every new round and deal 13 tiles to each of 4 players (seats East / South / West / North) plus reveal one dora indicator from the dead wall. When `--seed N` is supplied to `mahjong play`, the wall shuffle and all bot probabilistic decisions SHALL be deterministic — running the same seed twice produces a byte-identical sequence of dealt hands, draws, discards, calls, and outcomes. Without `--seed`, the system SHALL derive a seed from the OS PRNG and print it at game start.

When akadora is enabled (the default), the wall constructor SHALL substitute exactly one of the four copies of each five-rank tile (5m, 5p, 5s) with the red variant (`Tile{ID: tile.M5/P5/S5, Red: true}`) BEFORE the shuffle step, so deterministic seeds produce identical red-tile placements. When akadora is disabled, all 5-rank tiles SHALL be plain (`Red: false`). The wall MUST always contain exactly 4 copies of each tile by ID regardless of the akadora flag — substitution replaces a copy, it does not add or remove tiles.

The system SHALL expose a new constructor `NewWallWithOptions(seed int64, opts WallOptions) *Wall` that accepts `WallOptions{Akadora bool}`. The legacy `NewWall(seed int64)` SHALL continue to exist and SHALL delegate to `NewWallWithOptions(seed, WallOptions{Akadora: true})` so existing callers automatically get akadora-on, matching modern client conventions.

#### Scenario: Deterministic shuffle with explicit seed

- **GIVEN** the player runs `mahjong play --seed 42`
- **WHEN** the wall is shuffled and dealt
- **THEN** every player's initial 13-tile hand is identical to a previous run with the same seed
- **AND** the dora indicator is the same tile

#### Scenario: Random seed printed without explicit flag

- **GIVEN** the player runs `mahjong play` with no seed flag
- **WHEN** the game starts
- **THEN** the system prints a line of the form `Seed: <integer>` so the run can be reproduced
- **AND** subsequent runs with that exact integer via `--seed` produce the same game

#### Scenario: Each tile appears exactly four times

- **WHEN** any wall is constructed
- **THEN** each of the 34 tile types appears in exactly 4 copies, totalling 136 tiles

#### Scenario: Akadora-on wall contains exactly one red copy of each five

- **GIVEN** the player runs `mahjong play` (akadora default-on) with any seed
- **WHEN** the wall is constructed via `NewWall(seed)` or `NewWallWithOptions(seed, WallOptions{Akadora: true})`
- **THEN** the wall contains exactly one tile with `ID == tile.M5 && Red == true`, exactly one with `ID == tile.P5 && Red == true`, and exactly one with `ID == tile.S5 && Red == true`
- **AND** the wall contains exactly three plain copies of each five (`Red == false`)
- **AND** the total tile count is still 136

##### Example: red five counts under akadora-on

| ID       | Red==true count | Red==false count | Total |
| -------- | --------------- | ---------------- | ----- |
| tile.M5  | 1               | 3                | 4     |
| tile.P5  | 1               | 3                | 4     |
| tile.S5  | 1               | 3                | 4     |
| tile.M1  | 0               | 4                | 4     |

#### Scenario: Akadora-off wall contains no red tiles

- **GIVEN** the player runs `mahjong play --no-akadora --seed 42`
- **WHEN** the wall is constructed via `NewWallWithOptions(42, WallOptions{Akadora: false})`
- **THEN** the wall contains zero tiles with `Red == true`
- **AND** every five-rank tile is plain (`Red == false`)
- **AND** each tile ID still appears exactly 4 times

#### Scenario: Akadora substitution is deterministic under fixed seed

- **GIVEN** two wall constructions with `NewWallWithOptions(42, WallOptions{Akadora: true})`
- **WHEN** both walls are inspected tile-by-tile
- **THEN** the position of every tile (including red fives) is byte-identical between the two walls
