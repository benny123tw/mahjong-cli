## MODIFIED Requirements

### Requirement: Bot Decision Strategy

Bot opponents SHALL play a single hand-coded strategy with the following rules:

| Decision | Rule |
| -------- | ---- |
| Discard | Pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere). Honors and terminals score highest. Tiebreak: lowest tile ID. |
| Pon (yakuhai) | Always when bot has 2 copies of a discarded yakuhai tile (round wind, seat wind, or any dragon) |
| Pon (non-yakuhai) | 50% probability when bot has 2 copies AND bot is at shanten ≤ 2 |
| Chi | 40% probability, only from kamicha, only when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in bot's hand |
| Kan | Never |
| Riichi | When the bot's 14-tile hand has a discardable index that leaves a tenpai 13-tile shape AND the bot is concealed (no called melds) AND the bot's score is ≥1000 AND `Wall.LiveRemaining()` is ≥4. The bot SHALL pick the FIRST scanned index (0..len-1) whose post-discard hand has shanten=0 and submit `InputDiscard{Index: idx, Riichi: true}`. |
| Ron | When `calc.Analyze` on the bot's `concealed + discard` returns a non-nil result AND `Game.IsFuriten(seat)` returns false |
| Tsumo | When `calc.Analyze` on the bot's 14-tile hand (after drawing) returns a non-nil result |

All probabilistic decisions SHALL use a PRNG seeded from the same seed as the wall, so games reproduce deterministically. Bot riichi tile-choice is deterministic (first scanned index) and SHALL NOT consume from the PRNG.

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
- **AND** `Game.IsFuriten(bot)` returns false
- **WHEN** the claims window opens
- **THEN** the bot calls ron

#### Scenario: Bot ron blocked by permanent furiten

- **GIVEN** a bot's hand is tenpai on tile T but the bot's own pond contains T
- **WHEN** any opponent later discards T
- **THEN** the bot does NOT call ron — `Game.IsFuriten(bot)` is true and the dispatcher passes instead

#### Scenario: Bot tsumo on a yaku-bearing draw

- **GIVEN** a bot draws a tile completing a yaku-bearing winning shape (14-tile hand has `calc.Analyze` non-nil)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDeclareTsumo` and the round advances to `StateRoundOver{Outcome: OutcomeTsumo{...}}`

#### Scenario: Bot declares riichi when tenpai-after-discard

- **GIVEN** a bot's 14-tile hand has at least one discardable index that leaves a 13-tile shape with shanten=0
- **AND** the bot is concealed, score ≥1000, wall remaining ≥4
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDiscard{Index: <first-tenpai-leaving-index>, Riichi: true}`

#### Scenario: Bot does not declare riichi when open

- **GIVEN** a bot has previously called pon (one open meld)
- **AND** the bot's post-discard hand is tenpai
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot does NOT declare riichi (open hands cannot riichi); the bot falls back to the isolation-heuristic discard

#### Scenario: Bot does not declare riichi when wall has fewer than 4 tiles

- **GIVEN** the live wall has 3 tiles remaining
- **AND** a bot's post-discard hand is tenpai with concealed hand and ≥1000 score
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot does NOT declare riichi; the bot submits a regular `InputDiscard`

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
