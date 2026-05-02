# game-loop Specification

## Purpose

TBD - created by archiving change 'add-game-loop'. Update Purpose after archive.

## Requirements

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


<!-- @trace
source: add-akadora
updated: 2026-05-02
code:
  - internal/game/wall.go
  - internal/game/state.go
  - testdata/game/golden/seed-42.json
  - internal/game/kan.go
  - internal/game/call.go
  - internal/game/payout.go
  - internal/play/kan_keys.go
  - internal/play/play.go
  - cmd/play.go
  - internal/game/bot.go
  - internal/game/match.go
  - internal/game/turn.go
tests:
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/kan_test.go
  - internal/game/bot_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
-->

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
| Discard | When NO opponent has declared riichi, pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere). Honors and terminals score highest. Tiebreak: lowest tile ID. When AT LEAST ONE opponent has declared riichi AND the bot's 14-tile hand is at shanten ≤ 1 (push mode), pick the tile maximizing a danger-aware score = isolation - 2000 × danger, where danger is 0 for genbutsu (tile-ID matches any tile in a riichi-declarer's pond), 1 for suji-safe against a riichi-declarer (rank pair table — 1↔4, 7↔4, 2↔5, 8↔5, 3↔6, 9↔6 — same suit), and 2 otherwise. The 2000× constant guarantees any safe tile is preferred over any unsafe tile regardless of isolation difference. When AT LEAST ONE opponent has declared riichi AND the bot's 14-tile hand is at shanten ≥ 2 (fold mode), the bot SHALL pick the SAFEST tile by danger map regardless of isolation — the danger penalty constant SHALL be amplified to 1_000_000 (effectively infinite vs. the ~1000-range isolation score) so danger always dominates. Multi-riichi danger aggregates via MIN across declarers in both push and fold modes. |
| Pon (yakuhai) | Always when bot has 2 copies of a discarded yakuhai tile (round wind, seat wind, or any dragon) |
| Pon (non-yakuhai) | 50% probability when bot has 2 copies AND bot is at shanten ≤ 2 |
| Chi | 40% probability, only from kamicha, only when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in bot's hand |
| Kan | Never declares any flavor of kan (ankan, minkan, shouminkan). Bots MAY ron on a human's shouminkan upgrade tile via the chankan window — the existing ron path applies with `Chankan = true` populated by the engine. |
| Riichi | When the bot's 14-tile hand has a discardable index that leaves a tenpai 13-tile shape AND the bot is concealed (no called melds) AND the bot's score is ≥1000 AND `Wall.LiveRemaining()` is ≥4. The bot SHALL pick the FIRST scanned index (0..len-1) whose post-discard hand has shanten=0 and submit `InputDiscard{Index: idx, Riichi: true}`. |
| Ron | When `calc.Analyze` on the bot's `concealed + discard` returns a non-nil result AND `Game.IsFuriten(seat)` returns false (permanent OR temporary furiten blocks ron). Applies to both regular discards and shouminkan upgrade tiles surfaced via the chankan claim window. Fold mode does NOT block ron — if a winning tile lands, the bot still wins. |
| Tsumo | When `calc.Analyze` on the bot's 14-tile hand (after drawing) returns a non-nil result. Fold mode does NOT block tsumo. |

All probabilistic decisions SHALL use a PRNG seeded from the same seed as the wall, so games reproduce deterministically. Bot riichi tile-choice and danger-aware discard tile-choice (push and fold modes) are deterministic and SHALL NOT consume from the PRNG.

#### Scenario: Bot prefers genbutsu against riichi declarer

- **GIVEN** the human declared riichi and their pond contains `5p`
- **AND** a bot's hand contains both `5p` (genbutsu) and `9m` (unknown danger)
- **AND** the bot's isolation score for `9m` is higher than for `5p` in the absence of danger
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot discards `5p` (the genbutsu) — the danger penalty (-2000 × 2 = -4000) on `9m` outweighs the isolation difference

#### Scenario: Bot prefers suji over unknown when no genbutsu available

- **GIVEN** the human declared riichi and their pond contains `4p` (no other relevant tiles)
- **AND** a bot's hand contains `1p` (suji per the 1↔4 rule), `7p` (suji per the 7↔4 rule), and `3m` (unknown)
- **AND** none of these tiles are genbutsu (`4p` is in pond but the bot doesn't hold `4p`)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot discards `1p` or `7p` (both danger 1) over `3m` (danger 2), provided isolation scores don't extreme-favor `3m`. The danger gap (-2000 vs -4000) dominates a typical isolation gap

#### Scenario: Bot folds when shanten≥2 and opponent in riichi

- **GIVEN** the human declared riichi and their pond contains `5p` (genbutsu)
- **AND** a bot's 14-tile hand has shanten ≥ 2 (no realistic improvement path)
- **AND** the bot's hand contains `5p` (genbutsu, danger=0) and `1z` (unknown, danger=2) where `1z` has a much higher push-mode isolation score
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot discards `5p` — fold mode amplifies the danger penalty so the genbutsu is always picked over an unknown-danger tile regardless of isolation

##### Example: fold-mode discard with mixed danger levels

- **GIVEN** the bot's 14-tile hand at shanten=2: `1m 2m 4m 6m 8m 1p 3p 5p 7p 9p 1z 2z 3z 4z`
- **AND** the human's riichi pond contains `5p` and `1z`
- **AND** the danger map is `{5p: 0, 1z: 0, 4p: 1, 6p: 1, 2z: 2, 3z: 2, 4z: 2, ...}` (5p genbutsu, 1z genbutsu, 4p/6p suji, others unknown)
- **WHEN** the bot's fold-mode discard fires
- **THEN** the bot picks the index pointing to `5p` or `1z` (both danger=0; tiebreak by lowest tile ID picks `5p`), NOT the higher-isolation `4z` honor

#### Scenario: Bot in push mode keeps winning paths open

- **GIVEN** the human declared riichi and their pond contains `5p`
- **AND** a bot's 14-tile hand is at shanten = 1 (one tile away from tenpai)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot uses push-mode danger-aware scoring (K=2000) and may pick a moderately unsafe tile if the isolation score warrants — fold mode does NOT activate at shanten ≤ 1

#### Scenario: Bot ron blocked by temporary furiten

- **GIVEN** an opponent discarded a bot's machi tile T on a previous turn AND the bot did not ron on that discard
- **AND** before the bot's next own draw, a different opponent discards T again
- **WHEN** the bot's claim window evaluates ron
- **THEN** the bot does NOT submit ron — `Game.IsFuriten(bot)` returns true via the temporary-furiten arm and the dispatcher passes

#### Scenario: Bot does not chi from non-kamicha

- **GIVEN** a bot has 4m + 5m and the discarded 6m
- **WHEN** the discarder is not the bot's kamicha
- **THEN** the bot does not call chi regardless of dice roll

#### Scenario: Bot tsumo on a yaku-bearing draw

- **GIVEN** a bot draws a tile completing a yaku-bearing winning shape (14-tile hand has `calc.Analyze` non-nil)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDeclareTsumo` and the round advances to `StateRoundOver{Outcome: OutcomeTsumo{...}}`

#### Scenario: Bot declares riichi when tenpai-after-discard

- **GIVEN** a bot's 14-tile hand has at least one discardable index that leaves a 13-tile shape with shanten=0
- **AND** the bot is concealed, score ≥1000, wall remaining ≥4
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDiscard{Index: <first-tenpai-leaving-index>, Riichi: true}`

#### Scenario: Bot rons via chankan on human shouminkan

- **GIVEN** the human has an open `MeldPon` for `5p` and declares `InputDeclareShouminkan{TileID: tile.P5}`
- **AND** a bot is tenpai and not in furiten and the upgrade tile `5p` completes a yaku-bearing shape on the bot's hand
- **WHEN** the chankan window resolves
- **THEN** the bot submits `Claim{Kind: ClaimRon}` and the round terminates as `OutcomeRon{Winner: <bot>, Loser: <human>, Tile: 5p}` with `Chankan = true` in the winning context


<!-- @trace
source: add-bot-fold
updated: 2026-05-03
code:
  - internal/play/play.go
  - internal/game/bot.go
tests:
  - internal/game/bot_test.go
  - internal/play/play_test.go
-->

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
| Rinshan | Player wins by tsumo on a tile drawn from the dead wall after declaring kan (`Game.lastDrawWasRinshan[winner]` is true; cleared on next discard or call) |
| Chankan | Player wins by ron on a tile that an opponent just declared as shouminkan (engine entered `StateAwaitingChankan` and the winner's claim was honored) |
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

#### Scenario: Rinshan tsumo on kan replacement draw

- **GIVEN** a seat declares ankan and draws a rinshan replacement tile that completes their winning shape
- **WHEN** the seat submits `InputDeclareTsumo`
- **THEN** `Rinshan = true` is passed to `calc.Analyze` and rinshan kaihou is in the yaku list

#### Scenario: Chankan ron on shouminkan upgrade tile

- **GIVEN** an opponent declares shouminkan upgrading their open pon with the upgrade tile T
- **AND** the active seat is tenpai on T and not in furiten
- **WHEN** the active seat submits `Claim{Kind: ClaimRon}` in the chankan window
- **THEN** `Chankan = true` is passed to `calc.Analyze` and chankan is in the yaku list


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

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

---
### Requirement: Riichi Declaration

The system SHALL accept `InputDiscard{Riichi: true}` from a seat in `StateAwaitingDiscard` only when ALL of the following preconditions hold: the seat has zero called melds (concealed hand), the seat's score is at least 1000 points, the live wall has at least 4 tiles remaining, and the seat is in tenpai after the proposed discard. When all preconditions hold, the system SHALL deduct 1000 points from the seat as a riichi deposit, mark the seat as riichi-declared, open an ippatsu window for the seat, and complete the discard transition normally (advance to `StateAwaitingClaims`). When any precondition fails, the system SHALL return `ErrIllegalRiichi` and leave game state unchanged.

#### Scenario: Riichi declared on a tenpai concealed hand with funds

- **GIVEN** the human is in `StateAwaitingDiscard{Player: Human}` with a 14-tile concealed hand whose post-discard form is tenpai
- **AND** the human has 25000 points and the live wall has 60 tiles remaining
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the seat's score becomes 24000 (1000-point deposit deducted)
- **AND** the seat is marked riichi-declared
- **AND** state advances to `StateAwaitingClaims{Discarder: Human}`

#### Scenario: Riichi rejected when hand is open

- **GIVEN** the human has previously called pon (one open meld) and is in `StateAwaitingDiscard{Player: Human}`
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged (same hand, same state, no point deduction)

#### Scenario: Riichi rejected when wall has fewer than 4 tiles

- **GIVEN** the live wall has exactly 3 tiles remaining and the human is in `StateAwaitingDiscard{Player: Human}` with a tenpai post-discard hand
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged

#### Scenario: Riichi rejected when post-discard hand is not tenpai

- **GIVEN** the human is in `StateAwaitingDiscard{Player: Human}` and `hand.Shanten` of the post-discard 13-tile hand is ≥1
- **WHEN** the system receives `InputDiscard{Index: 5, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged

#### Scenario: Riichi rejected when seat has fewer than 1000 points

- **GIVEN** the human's score is 800 (e.g., after prior noten payments)
- **AND** the human is in `StateAwaitingDiscard{Player: Human}` on a tenpai post-discard hand
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`

##### Example: deposit deduction at declaration

- **GIVEN** human score = 25000, wall remaining = 60, hand = `[1m,2m,3m,4m,5m,6m,7m,8m,9m,1p,1p,2p,2p, drawn=3p]` (chiitoitsu-no, but the post-discard hand `[1m..9m, 1p, 1p, 2p, 2p]` is shanten=1 — pinfu wait... actually let's pick a valid tenpai)
- **WHEN** the human discards `9m` with `Riichi: true` from a hand that lands at tenpai
- **THEN** post-state: score=24000, riichiDeclared[Human]=true, ippatsuLive[Human]=true


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Riichi-Restricted Discard

After a seat has declared riichi, the system SHALL only accept `InputDiscard` from that seat when the discard index points to the just-drawn tile (the rightmost tile in the seat's hand, at index `len(hand)-1`). Any `InputDiscard` with a different index from a riichi-declared seat SHALL return `ErrIllegalDiscard` and leave game state unchanged. This restriction applies starting on the seat's NEXT turn after the riichi-declaring discard (the declaring discard itself is selected freely; subsequent discards are forced).

#### Scenario: Post-riichi discard locked to drawn tile

- **GIVEN** the human declared riichi on a previous turn and is now in `StateAwaitingDiscard{Player: Human}` with a 14-tile hand
- **WHEN** the system receives `InputDiscard{Index: 5}` (a sorted-hand tile, not the drawn tile)
- **THEN** the system returns `ErrIllegalDiscard`

#### Scenario: Post-riichi discard accepted at index 13

- **GIVEN** the human declared riichi on a previous turn and is now in `StateAwaitingDiscard{Player: Human}` with a 14-tile hand
- **WHEN** the system receives `InputDiscard{Index: 13}` (the just-drawn tile)
- **THEN** the discard transition completes normally (advance to `StateAwaitingClaims`)


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Ippatsu Window Tracking

The system SHALL track an ippatsu window for each seat that has declared riichi. The window opens at the moment the seat's riichi-declaring discard transitions to `StateAwaitingClaims`. The window closes when EITHER (a) the seat makes their next non-riichi discard since declaration (i.e., they drew and chose not to win), OR (b) any seat (including the declarer) executes a successful pon or chi call. While the window is open, the seat may either ron on any opponent's discard or tsumo on their own next draw and earn ippatsu. When the seat wins while the window is open, the system SHALL pass `Ippatsu = true` to `calc.Analyze` via `Game.contextForWin`; otherwise `Ippatsu = false`.

#### Scenario: Ippatsu when riichi → opponents pass → win on own next draw

- **GIVEN** the human declares riichi and the next state is `StateAwaitingClaims{Discarder: Human}` with no claimants
- **AND** West / North / East all draw and discard with no calls intervening
- **WHEN** the human draws their next tile and declares tsumo
- **THEN** `calc.Context{Ippatsu: true}` is passed to `calc.Analyze`

#### Scenario: Ippatsu broken by intervening pon

- **GIVEN** the human declares riichi
- **AND** before the human's next draw, West calls pon on East's discard
- **WHEN** the human eventually wins (tsumo or ron)
- **THEN** `calc.Context{Ippatsu: false}` is passed to `calc.Analyze`

#### Scenario: Ippatsu closes on the seat's next non-tsumo discard

- **GIVEN** the human declares riichi on turn 5 and the ippatsu window is open
- **AND** the round proceeds with no calls; the human's draw on turn 6 happens but they do not tsumo
- **AND** the human discards the drawn tile (forced by riichi-restricted-discard)
- **WHEN** the human eventually wins on a later turn
- **THEN** `calc.Context{Ippatsu: false}` is passed to `calc.Analyze` (window closed at turn 6 own-discard)


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Double Riichi Detection

When `InputDiscard{Riichi: true}` succeeds AND the declaring seat has not yet discarded any tile this round AND no calls have happened this round, the system SHALL mark the declaration as a "double riichi". When the seat subsequently wins, `calc.Context{DoubleRiichi: true}` SHALL be passed to `calc.Analyze`. The standard `Riichi` flag SHALL remain set as well; the calc layer dedupes (existing yaku-detection contract).

#### Scenario: Double riichi on first uninterrupted intake

- **GIVEN** the dealer (East) draws their first tile and the round has zero prior discards and no prior calls
- **AND** the dealer's post-discard hand is tenpai
- **WHEN** the dealer submits `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the seat's `doubleRiichi[East] = true` is recorded
- **AND** when the dealer subsequently wins, `calc.Context.DoubleRiichi = true` is passed

#### Scenario: Riichi declared after any discard is regular riichi only

- **GIVEN** East draws and discards (regular discard, no riichi) on turn 1
- **AND** South draws on turn 2 and decides to declare riichi
- **WHEN** South submits `InputDiscard{Index: 13, Riichi: true}`
- **THEN** `doubleRiichi[South] = false` and `riichiDeclared[South] = true`
- **AND** when South wins, `calc.Context.Riichi = true, DoubleRiichi = false`


<!-- @trace
source: add-human-agari
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/play/play.go
  - internal/game/state.go
tests:
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/riichi_test.go
-->

---
### Requirement: Furiten Query

The system SHALL expose `Game.IsFuriten(seat Seat) bool` returning true when EITHER permanent furiten OR temporary furiten is active for the seat. Permanent furiten holds when any tile in the seat's own discard pond matches a tile ID in the seat's current machi (computed via `hand.Machi` on the seat's concealed hand at exactly 13 tiles). Temporary furiten holds when an opponent has discarded a tile that completes a winning shape on the seat's hand (`hand.IsWinning(concealed + discard) == true`) since the seat's last own draw, AND the seat did not submit a ron claim for that discard. Temporary furiten clears when the seat takes their next own draw (`stepFromAwaitingDraw` resets the per-seat flag). When the seat's hand is not exactly 13 tiles, `IsFuriten` SHALL return false (the machi is undefined for non-tenpai shapes, and yaku-less winning shapes still trigger the temporary lockout but require a 13-tile concealed hand to be meaningful).

#### Scenario: Permanent furiten when machi tile is in own pond

- **GIVEN** the seat's 13-tile hand has machi `{4m, 7m}` and the seat's discard pond contains `4m`
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `true`

#### Scenario: Not furiten when machi tiles are absent and no winning tile passed

- **GIVEN** the seat's 13-tile hand has machi `{4m, 7m}`, the seat's pond contains `1z, 9m, 5p`, and no opponent has discarded `4m` or `7m` since the seat's last draw
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `false`

#### Scenario: Temporary furiten when opponent discards machi tile and seat passes

- **GIVEN** the seat's 13-tile hand wins on `5p` and the seat is NOT in permanent furiten
- **AND** an opponent discards `5p` and the seat does not submit a ron claim
- **WHEN** `Game.IsFuriten(seat)` is called before the seat's next draw
- **THEN** the result is `true` (temporary furiten armed)

#### Scenario: Temporary furiten clears on the seat's next own draw

- **GIVEN** the seat is in temporary furiten (machi tile passed since last draw)
- **WHEN** the seat takes their next own draw via `stepFromAwaitingDraw`
- **THEN** `Game.IsFuriten(seat)` returns false (temporary flag reset; permanent furiten still applies if relevant)

#### Scenario: Furiten query on non-tenpai hand returns false

- **GIVEN** the seat's 13-tile hand has shanten ≥1 (machi is empty)
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `false`


<!-- @trace
source: add-bot-defense
updated: 2026-05-02
code:
  - internal/game/bot.go
  - internal/game/turn.go
  - internal/play/play.go
tests:
  - internal/game/bot_test.go
  - internal/game/furiten_test.go
  - internal/play/play_test.go
-->

---
### Requirement: Human Ron From Claim Window

The system SHALL accept `InputResolveClaims{Claims: {seat: Claim{Kind: ClaimRon}}}` in `StateAwaitingClaims` from any non-discarder seat when ALL of the following hold: `calc.Analyze` on the seat's `concealed + discard` returns a non-nil result, AND `Game.IsFuriten(seat)` returns false. The transition SHALL go through the existing `stepFromAwaitingClaims` ron path: build the winning `hand.Hand`, call `calc.Analyze` with the populated `contextForWin`, and transition to `StateRoundOver{Outcome: OutcomeRon{...}}`. When `calc.Analyze` returns nil (no yaku) OR `IsFuriten(seat)` is true, the system SHALL return `ErrYakulessWin` or `ErrFuritenRon` respectively, and leave game state unchanged. The furiten gate SHALL apply to every seat — the previous human-only restriction is removed; bots are subject to the same permanent-furiten rule.

#### Scenario: Human ron on a yaku-bearing discard

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the human's `concealed + 5p` forms a yaku-bearing winning shape
- **AND** `Game.IsFuriten(Human)` returns false
- **WHEN** the system receives `InputResolveClaims{Claims: {Human: ClaimRon}}`
- **THEN** state advances to `StateRoundOver{Outcome: OutcomeRon{Winner: Human, Loser: East, ...}}`

#### Scenario: Human ron rejected when furiten

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the human's hand would win on 5p but the human's own pond contains 5p
- **WHEN** the system receives `InputResolveClaims{Claims: {Human: ClaimRon}}`
- **THEN** the system returns `ErrFuritenRon`
- **AND** game state is unchanged

#### Scenario: Bot ron on a yaku-bearing discard

- **GIVEN** a bot at SeatNorth is in tenpai with a yaku-bearing wait on 5p
- **AND** `Game.IsFuriten(SeatNorth)` returns false
- **WHEN** an opponent discards 5p and `InputResolveClaims{Claims: {SeatNorth: ClaimRon}}` is submitted
- **THEN** state advances to `StateRoundOver{Outcome: OutcomeRon{Winner: SeatNorth, ...}}`

#### Scenario: Bot ron rejected when furiten

- **GIVEN** a bot at SeatNorth has a tenpai hand whose machi includes 5p
- **AND** SeatNorth's own pond contains 5p (permanent furiten)
- **WHEN** an opponent discards 5p and `InputResolveClaims{Claims: {SeatNorth: ClaimRon}}` is submitted
- **THEN** the system returns `ErrFuritenRon` and game state is unchanged

<!-- @trace
source: add-smart-ai
updated: 2026-05-02
code:
  - internal/game/bot.go
  - internal/play/play.go
  - internal/game/turn.go
tests:
  - internal/game/bot_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
-->

---
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

<!-- @trace
source: add-multi-round
updated: 2026-05-02
code:
  - testdata/game/golden/seed-42.json
  - cmd/play.go
  - internal/play/play.go
  - internal/game/match.go
  - internal/game/bot.go
  - internal/game/turn.go
  - internal/game/payout.go
tests:
  - internal/game/bot_test.go
  - internal/game/match_test.go
  - internal/game/turn_test.go
  - internal/play/play_test.go
  - internal/game/furiten_test.go
  - internal/game/payout_test.go
-->