## MODIFIED Requirements

### Requirement: Bot Decision Strategy

Bot opponents SHALL play a single hand-coded strategy with the following rules:

| Decision | Rule |
| -------- | ---- |
| Discard | When NO opponent has declared riichi, pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere). Honors and terminals score highest. Tiebreak: lowest tile ID. When AT LEAST ONE opponent has declared riichi, pick the tile maximizing a danger-aware score = isolation - 2000 × danger, where danger is 0 for genbutsu (tile-ID matches any tile in a riichi-declarer's pond), 1 for suji-safe against a riichi-declarer (rank pair table — 1↔4, 7↔4, 2↔5, 8↔5, 3↔6, 9↔6 — same suit), and 2 otherwise. The 2000× constant guarantees any safe tile is preferred over any unsafe tile regardless of isolation difference. When multiple riichi declarers exist, danger is the MIN across them (the safest declaration determines the score). |
| Pon (yakuhai) | Always when bot has 2 copies of a discarded yakuhai tile (round wind, seat wind, or any dragon) |
| Pon (non-yakuhai) | 50% probability when bot has 2 copies AND bot is at shanten ≤ 2 |
| Chi | 40% probability, only from kamicha, only when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in bot's hand |
| Kan | Never declares any flavor of kan (ankan, minkan, shouminkan). Bots MAY ron on a human's shouminkan upgrade tile via the chankan window — the existing ron path applies with `Chankan = true` populated by the engine. |
| Riichi | When the bot's 14-tile hand has a discardable index that leaves a tenpai 13-tile shape AND the bot is concealed (no called melds) AND the bot's score is ≥1000 AND `Wall.LiveRemaining()` is ≥4. The bot SHALL pick the FIRST scanned index (0..len-1) whose post-discard hand has shanten=0 and submit `InputDiscard{Index: idx, Riichi: true}`. |
| Ron | When `calc.Analyze` on the bot's `concealed + discard` returns a non-nil result AND `Game.IsFuriten(seat)` returns false (permanent OR temporary furiten blocks ron). Applies to both regular discards and shouminkan upgrade tiles surfaced via the chankan claim window. |
| Tsumo | When `calc.Analyze` on the bot's 14-tile hand (after drawing) returns a non-nil result |

All probabilistic decisions SHALL use a PRNG seeded from the same seed as the wall, so games reproduce deterministically. Bot riichi tile-choice and danger-aware discard tile-choice are deterministic and SHALL NOT consume from the PRNG.

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
