## MODIFIED Requirements

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
