## ADDED Requirements

### Requirement: Hanchan Match Structure

The system SHALL model a hanchan match as exactly 8 hands indexed 0..7: indices 0-3 are the East round (round wind = `tile.EastWind`), and indices 4-7 are the South round (round wind = `tile.SouthWind`). The match SHALL be created via `match.NewMatch(seed int64) *Match` initializing all four seats' scores to 25000, dealer = `SeatEast`, hand index = 0, honba = 0, riichi sticks = 0, round wind = `tile.EastWind`, and the active per-hand `*Game` constructed via `game.NewWithDealer(seed, SeatEast, tile.EastWind)`.

#### Scenario: Fresh match starts at East 1 with all seats at 25000

- **GIVEN** `match.NewMatch(7)` is called
- **WHEN** the caller queries `Match.Scores()`, `Match.Dealer()`, `Match.HandIndex()`, `Match.Honba()`, `Match.RoundWind()`
- **THEN** scores are `[25000, 25000, 25000, 25000]`, dealer is `SeatEast`, hand index is 0, honba is 0, round wind is `tile.EastWind`

---

### Requirement: Match Advancement From Outcome

The system SHALL expose `Match.AdvanceFromOutcome(o Outcome) (TransitionResult, error)` that consumes the active hand's terminal outcome (one of `OutcomeRon`, `OutcomeTsumo`, `OutcomeRyuukyoku`) and produces a `TransitionResult` describing what happened. The function SHALL: (1) compute per-seat score deltas via `ComputePayouts`, (2) apply deltas to `Match.scores`, (3) apply riichi-stick pool changes, (4) detect renchan vs rotation and update `dealer`/`handIndex`/`honba`/`roundWind` accordingly, (5) check end-of-match conditions (handIndex past South 4 with no renchan, or any seat below 0), and (6) construct the next hand's `*Game` (or set `Match.outcome != nil` when the match has finished). The returned `TransitionResult` SHALL include the score deltas, the new totals, the renchan flag, the new hand index, and an optional `MatchOutcome` populated when the hanchan ends.

#### Scenario: Non-dealer ron rotates dealer and resets honba

- **GIVEN** a fresh match at East 1, dealer = `SeatEast`, honba = 0
- **WHEN** `Match.AdvanceFromOutcome(OutcomeRon{Winner: SeatSouth, Loser: SeatEast, ...})` is called with a 30fu 1han ron Award
- **THEN** the returned TransitionResult has `Renchan = false`, `NewHandIndex = 1`
- **AND** `Match.Dealer()` returns `SeatSouth`, `Match.HandIndex()` returns 1, `Match.Honba()` returns 0

#### Scenario: Dealer tsumo triggers renchan and increments honba

- **GIVEN** a match at East 1, dealer = `SeatEast`, honba = 0
- **WHEN** `Match.AdvanceFromOutcome(OutcomeTsumo{Winner: SeatEast, ...})` is called with any winning Award
- **THEN** the returned TransitionResult has `Renchan = true`, `NewHandIndex = 0`
- **AND** `Match.Dealer()` returns `SeatEast`, `Match.HandIndex()` returns 0, `Match.Honba()` returns 1

#### Scenario: Dealer-tenpai ryuukyoku is renchan

- **GIVEN** a match at East 1, dealer = `SeatEast`, honba = 0
- **WHEN** `Match.AdvanceFromOutcome(OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatNorth}})` is called
- **THEN** the returned TransitionResult has `Renchan = true`
- **AND** `Match.Honba()` returns 1, `Match.Dealer()` is unchanged

#### Scenario: Round wind transitions East to South after East 4

- **GIVEN** a match at East 4 (handIndex = 3), dealer = `SeatNorth`, with a non-renchan outcome
- **WHEN** `Match.AdvanceFromOutcome(OutcomeRon{Winner: SeatSouth, Loser: SeatEast, ...})` is called
- **THEN** `Match.HandIndex()` returns 4, `Match.RoundWind()` returns `tile.SouthWind`, `Match.Dealer()` returns `SeatEast` (rotated from North → East)

---

### Requirement: Score Payout Computation

The system SHALL expose `ComputePayouts(o Outcome, ctx PayoutContext) [4]int` returning per-seat point deltas (positive = received, negative = paid). `PayoutContext` SHALL carry `Dealer Seat`, `Honba int`, and `RiichiSticks int`. For each outcome:

| Outcome | Formula |
| ------- | ------- |
| `OutcomeRon{Winner, Loser, Result}` | Winner gains `Result.Award.Total + 300*Honba + 1000*RiichiSticks`. Loser pays `Result.Award.Total + 300*Honba`. The riichi-stick pool delta is `-RiichiSticks` (winner sweeps). |
| `OutcomeTsumo{Winner, Result}` | Per `Award.Base`: dealer winner → each non-dealer pays `roundUp100(Base*2) + 100*Honba`; non-dealer winner → each non-dealer pays `roundUp100(Base*1) + 100*Honba`, dealer pays `roundUp100(Base*2) + 100*Honba`. Winner receives the sum + `1000*RiichiSticks`. |
| `OutcomeRyuukyoku{TenpaiPlayers}` | If 0 or 4 tenpai → all deltas 0. Otherwise total transfer is 3000: each noten seat pays `3000 / NotenCount`, each tenpai seat receives `3000 / TenpaiCount`. Honba does NOT add to ryuukyoku transfers. Riichi sticks pool stays unchanged. |

The function SHALL NOT mutate state. The caller (`Match.AdvanceFromOutcome`) applies the returned deltas.

#### Scenario: Non-dealer ron 30fu 3han with honba 2

- **GIVEN** a non-dealer winning hand with `Award.Total = 3900` and `Award.Base = 960` (30fu × 2^(2+3))
- **AND** `PayoutContext{Dealer: SeatEast, Honba: 2, RiichiSticks: 0}`, winner = `SeatSouth`, loser = `SeatNorth`
- **WHEN** `ComputePayouts(OutcomeRon{Winner: SeatSouth, Loser: SeatNorth, Result: ...})` is called
- **THEN** the returned delta for `SeatSouth` is `+4500` (3900 + 600 honba), for `SeatNorth` is `-4500`, for `SeatEast` and `SeatWest` is `0`

#### Scenario: Dealer tsumo mangan with honba 0

- **GIVEN** a dealer-tsumo win with `Award.Base = 2000` (mangan), `PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 1}`, winner = `SeatEast`
- **WHEN** `ComputePayouts(OutcomeTsumo{Winner: SeatEast, Result: ...})` is called
- **THEN** the returned delta for `SeatEast` is `+13000` (4000 each from 3 non-dealers + 1 stick × 1000), for each non-dealer is `-4000`

#### Scenario: Ryuukyoku with 1 tenpai pays 3000 from each noten

- **GIVEN** `OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatNorth}}`, `PayoutContext{Dealer: SeatEast, Honba: 0, RiichiSticks: 0}`
- **WHEN** `ComputePayouts(...)` is called
- **THEN** the delta for `SeatNorth` is `+3000`, for each of `SeatEast`/`SeatSouth`/`SeatWest` is `-1000`

#### Scenario: Ryuukyoku with all 4 tenpai produces zero transfer

- **GIVEN** `OutcomeRyuukyoku{TenpaiPlayers: []Seat{SeatEast, SeatSouth, SeatWest, SeatNorth}}`
- **WHEN** `ComputePayouts(...)` is called
- **THEN** all four returned deltas are `0`

---

### Requirement: Match End Conditions

The match SHALL end on either of two conditions:

1. **Hanchan completion**: `handIndex` would advance past 7 (i.e., a non-renchan outcome occurs at South 4) → `MatchOutcomeFinished{Reason: "hanchan-complete"}`.
2. **Tobi (bust)**: After applying payouts, any seat's score is below 0 → `MatchOutcomeFinished{Reason: "tobi", BustSeat: <seat>}`.

When the match has ended, `Match.IsFinished()` returns true and `Match.FinalOutcome()` returns the populated `MatchOutcome`. Subsequent calls to `Match.AdvanceFromOutcome` SHALL return an error `ErrMatchAlreadyFinished` and not mutate state.

#### Scenario: Hanchan completes after South 4 non-renchan ron

- **GIVEN** a match at South 4 (handIndex = 7), dealer = `SeatNorth`, honba = 0, with no busted seats
- **WHEN** a non-renchan outcome resolves (e.g., `OutcomeRon{Winner: SeatSouth, Loser: SeatEast, ...}`)
- **THEN** `Match.IsFinished()` returns true
- **AND** `Match.FinalOutcome().Reason` is `"hanchan-complete"`

#### Scenario: Tobi ends the match mid-hanchan

- **GIVEN** a match at East 2 with `SeatNorth` at 1500 and a haneman dealer ron about to fire from `SeatEast`
- **WHEN** the dealer ron resolves with `Award.Total = 18000` and `SeatNorth` is the loser
- **THEN** `SeatNorth`'s post-payout score is below 0
- **AND** `Match.IsFinished()` returns true with `FinalOutcome().Reason = "tobi"` and `BustSeat = SeatNorth`

---

### Requirement: Per-Hand Game Construction Determinism

For replay determinism, the system SHALL seed each per-hand `*Game` as `game.NewWithDealer(matchSeed + int64(handIndex), dealer, roundWind)`. Across renchan repetitions, the seed advances by 0 (same hand index = same seed). Across rotations, the seed advances by 1 (new hand index = new seed). Logging the per-hand effective seed SHALL be sufficient for any single-hand bug to be reproduced via `game.NewWithDealer(seed, dealer, roundWind)` outside of a Match.

#### Scenario: Same handIndex on renchan replays the same wall

- **GIVEN** a match at East 1 (handIndex = 0, seed = 7) with a dealer-tsumo renchan that bumps honba to 1
- **WHEN** the next hand is constructed
- **THEN** the new `*Game`'s seed-derived wall is identical to the original (seed = 7 + 0 = 7) — the wall shuffle is reproducible

#### Scenario: Rotation advances per-hand seed

- **GIVEN** a match at East 1 (handIndex = 0, seed = 7) with a non-renchan outcome
- **WHEN** the next hand is constructed
- **THEN** the new `*Game` is built with seed = 8 (`matchSeed + handIndex` = `7 + 1`)
