# game-loop Specification

## Purpose

TBD - created by archiving change 'add-game-loop'. Update Purpose after archive.

## Requirements

### Requirement: Wall Construction and Dealing

The system SHALL construct a 136-tile wall (4 copies each of 34 tile types) for every new round and deal 13 tiles to each of 4 players (seats East / South / West / North) plus reveal one dora indicator from the dead wall. When `--seed N` is supplied to `mahjong play`, the wall shuffle and all bot probabilistic decisions SHALL be deterministic — running the same seed twice produces a byte-identical sequence of dealt hands, draws, discards, calls, and outcomes. Without `--seed`, the system SHALL derive a seed from the OS PRNG and print it at game start.

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
- **AND** the wall contains no red-five tiles in v1 (red fives ship with the akadora-toggle change)

---
### Requirement: Turn Cycle State Machine

The system SHALL drive game progression as a state machine over five named states: `AwaitingDraw{Player}`, `AwaitingDiscard{Player}`, `AwaitingClaims{Discard, Discarder}`, `RoundOver{Outcome}`, and `GameOver{Standings}`. Transitions SHALL be pure functions of the current state and a single event, returning a new state plus zero or more emitted events. Direct mutation of state from outside the transition functions SHALL NOT be possible.

#### Scenario: Draw advances to AwaitingDiscard

- **GIVEN** state is `AwaitingDraw{Player: South}` and the wall has at least one tile remaining
- **WHEN** the South player draws a tile
- **THEN** state becomes `AwaitingDiscard{Player: South}`
- **AND** the South player's hand has 14 tiles

#### Scenario: Discard with no claims advances to next player's draw

- **GIVEN** state is `AwaitingDiscard{Player: South}`
- **WHEN** South discards tile T and no other player calls within the claims window
- **THEN** state passes through `AwaitingClaims{Discard: T, Discarder: South}` and resolves to `AwaitingDraw{Player: West}`
- **AND** the discarded tile T is appended to South's pond record

#### Scenario: Wall exhausted with no winning hand transitions to RoundOver via ryuukyoku

- **GIVEN** state is `AwaitingDraw{Player: any}` and the wall has zero remaining drawable tiles
- **WHEN** the draw is attempted
- **THEN** state becomes `RoundOver{Outcome: Ryuukyoku{TenpaiPlayers: [...]}}` with the list of players currently in tenpai

---
### Requirement: Call Resolution Priority

When a player discards a tile, the system SHALL collect every legal claim from the other three players within a single claims window, then resolve them in fixed priority order: ron beats pon, pon beats chi, and ties on ron use the head-bump rule (the player closest to the discarder going right-around-the-table wins; other ron claims are not paid out). At most one player wins the claim per discard.

#### Scenario: Ron beats pon

- **GIVEN** South discards 5p
- **AND** West can pon 5p (has two 5p), North can ron on 5p (winning hand with yaku)
- **WHEN** the claims window resolves
- **THEN** North wins the claim with ron
- **AND** West's pon is not executed
- **AND** the round transitions to `RoundOver{Outcome: Ron{Winner: North, Loser: South, ...}}`

#### Scenario: Pon beats chi

- **GIVEN** West discards 4m
- **AND** North can pon 4m (has two 4m), East can chi 4m (kamicha of West, has 2m+3m or 5m+6m)
- **WHEN** the claims window resolves
- **THEN** North wins with pon

#### Scenario: Chi only legal from kamicha

- **GIVEN** South discards 4m
- **AND** East has 2m+3m (could chi if legal), West has 5m+6m (could chi if legal)
- **WHEN** the claims window resolves
- **THEN** West wins chi (West is kamicha-of-South in seat order East→South→West→North) — East cannot chi a discard from South

#### Scenario: Head-bump on competing ron claims

- **GIVEN** East discards 7s
- **AND** West and North can both ron on 7s
- **WHEN** the claims window resolves
- **THEN** West wins (West is closer to East going right around the table) and North's ron is not paid out

---
### Requirement: Bot Decision Strategy

Bot opponents SHALL play a single hand-coded strategy with the following rules:

| Decision | Rule |
| -------- | ---- |
| Discard | Pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere). Honors and terminals score highest. Tiebreak: lowest tile ID. |
| Pon (yakuhai) | Always when bot has 2 copies of a discarded yakuhai tile (round wind, seat wind, or any dragon) |
| Pon (non-yakuhai) | 50% probability when bot has 2 copies AND bot is at shanten ≤ 2 |
| Chi | 40% probability, only from kamicha, only when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in bot's hand |
| Kan | Never |
| Riichi | Never |
| Ron | Always when discarded tile completes a yaku-bearing winning hand |
| Tsumo | Always when drawn tile completes a yaku-bearing winning hand |

All probabilistic decisions SHALL use a PRNG seeded from the same seed as the wall, so games reproduce deterministically.

#### Scenario: Bot pons yakuhai always

- **GIVEN** a bot at seat North in round East has two East-wind tiles in hand
- **WHEN** any opponent discards East
- **THEN** the bot calls pon (probability 1.0)

#### Scenario: Bot does not chi from non-kamicha

- **GIVEN** a bot has 4m + 5m and the discarded 6m
- **WHEN** the discarder is not the bot's kamicha
- **THEN** the bot does not call chi regardless of dice roll

#### Scenario: Bot ron on yaku-bearing discard

- **GIVEN** a bot's hand is tenpai with at least one possible yaku and an opponent discards a winning tile
- **WHEN** the claims window opens
- **THEN** the bot calls ron

#### Scenario: Bot does not declare riichi in v1

- **WHEN** any bot reaches tenpai during a hand
- **THEN** no riichi declaration is emitted regardless of hand quality

---
### Requirement: Round Termination and Outcome

A round SHALL terminate by exactly one of two outcomes: an agari (a player completes a winning hand by tsumo or ron) or a ryuukyoku (the wall exhausts before any agari). On agari, the system SHALL invoke `calc.Analyze` with the winning hand and a fully-populated context (seat, round wind, riichi state, dora indicators, and the eight Group C state flags) to produce the score award. On ryuukyoku, the system SHALL identify which players are in tenpai for the noten payment exchange (1500 from each noten player split equally among tenpai players, capped at 3000 per tenpai player).

#### Scenario: Tsumo agari produces a scored result

- **GIVEN** a player draws a tile that completes their hand with at least one yaku
- **WHEN** they declare tsumo
- **THEN** state becomes `RoundOver{Outcome: Tsumo{Winner, Hand, Award}}` with `Award` populated by `calc.Analyze`

#### Scenario: Ryuukyoku noten payments

- **GIVEN** the wall exhausts and 2 players are tenpai, 2 are noten
- **WHEN** the round ends
- **THEN** each noten player pays 1500
- **AND** each tenpai player receives 1500 (3000 split between 2 tenpai players)

#### Scenario: Yakuless win is not allowed

- **GIVEN** a player reaches a winning shape with no yaku (e.g., open hand of all simples without tanyao yakuhai or any other yaku)
- **WHEN** they would attempt tsumo or ron
- **THEN** the system MUST NOT advance to `RoundOver` and SHALL keep the player in their current state — the win is not legal

---
### Requirement: Group C Game Context Flags

The game state machine SHALL track and populate eight contextual flags whenever a winning hand is being scored, by passing them into `yaku.Context` via `calc.Analyze`:

| Flag | Set when |
| ---- | -------- |
| Ippatsu | Player wins within one turn after declaring riichi without any calls (own or opponents') intervening |
| Haitei | Player wins by tsumo on the very last drawable tile of the live wall |
| Houtei | Player wins by ron on the very last discard of a hand that exhausted the wall |
| Rinshan | Player wins by tsumo on a tile drawn from the dead wall after declaring kan (always false in v1, kan deferred) |
| Chankan | Player wins by ron on a tile that an opponent just declared as added-kan (always false in v1, kan deferred) |
| DoubleRiichi | Player declared riichi on their first uninterrupted draw — no calls between deal and that draw |
| Tenhou | Dealer wins on the initial 14-tile dealt hand (no draws, no discards happened yet) |
| Chiihou | Non-dealer wins on their first draw with no calls intervening |

#### Scenario: Ippatsu when riichi → no calls → win

- **GIVEN** the South player declares riichi on turn 5 and discards
- **AND** no player makes any call before South's next draw
- **WHEN** South draws a winning tile and declares tsumo
- **THEN** `Ippatsu = true` is passed to `calc.Analyze` and ippatsu is in the yaku list

#### Scenario: Ippatsu broken by an intervening call

- **GIVEN** the South player declares riichi on turn 5
- **AND** West calls pon on East's discard before South's next draw
- **WHEN** South later wins
- **THEN** `Ippatsu = false` and ippatsu is not in the yaku list

#### Scenario: Haitei tsumo on the last live tile

- **GIVEN** the wall has exactly one drawable tile remaining
- **WHEN** the next player draws it and declares tsumo on it
- **THEN** `Haitei = true` and haitei raoyue is in the yaku list

#### Scenario: Tenhou for dealer's initial hand

- **GIVEN** the wall is dealt and the dealer's 14-tile hand (13 dealt + 14th drawn first because dealer draws first) forms a winning shape
- **WHEN** the dealer declares tsumo before discarding
- **THEN** `Tenhou = true` and tenhou is in the yaku list (yakuman)

---
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