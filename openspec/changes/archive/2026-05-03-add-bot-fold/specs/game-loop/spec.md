## MODIFIED Requirements

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
