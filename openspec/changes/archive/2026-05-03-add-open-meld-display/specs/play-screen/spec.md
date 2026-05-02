## ADDED Requirements

### Requirement: Open Meld Display For Human Player

The system SHALL render the human player's called melds (pon, chi, ankan, minkan, shouminkan) to the right of the concealed hand row, separated from the concealed tiles by a 2-tile-width gap. Each meld SHALL be rendered as a contiguous group of its constituent tiles (3 for pon/chi, 4 for any kan flavor) prefixed with a textual seat-source marker, and adjacent melds SHALL be separated by a single-tile-width gap.

The seat-source marker SHALL be `[E]`, `[S]`, `[W]`, or `[N]` for melds whose called tile came from East / South / West / North respectively (pon, chi, minkan, shouminkan), and SHALL be `[A]` for ankan (concealed kan, no source seat). The marker uses literal ASCII brackets and a single seat letter in BOTH the Unicode and ASCII renderers — no glyph rotation, no per-renderer divergence in the marker convention.

When the human has zero open melds, the rendered output SHALL be byte-identical to the pre-change rendering of `renderHand()` — the open-meld block contributes nothing.

When the combined width of the concealed-hand block plus the open-meld block exceeds 80 columns, the open-meld block SHALL wrap onto a second line directly below the concealed-hand row rather than truncating or pushing the concealed row off-screen.

#### Scenario: Pon meld renders with East source marker

- **GIVEN** the human's open-melds list contains a single `MeldPon{Tiles: [5p, 5p, 5p], From: SeatEast}`
- **WHEN** the play screen renders
- **THEN** the rendered output contains the substring `[E]` followed by three `5p` tile glyphs, positioned to the right of the concealed-hand row

#### Scenario: Ankan meld renders with the [A] marker

- **GIVEN** the human's open-melds list contains a single `MeldKan{KanKind: KanAnkan, Tiles: [1m, 1m, 1m, 1m]}`
- **WHEN** the play screen renders
- **THEN** the rendered output contains the substring `[A]` followed by four `1m` tile glyphs

##### Example: three melds rendered side-by-side

- **GIVEN** the human's open-melds list is `[MeldPon{5p, From: East}, MeldChi{2m,3m,4m, From: West}, MeldKan{Ankan, 1z×4}]`
- **WHEN** the play screen renders
- **THEN** the rendered output contains all three meld blocks separated by single-tile-width gaps, in order: `[E] 5p 5p 5p`, `[W] 2m 3m 4m`, `[A] 1z 1z 1z 1z`

#### Scenario: No open melds preserves prior rendering

- **GIVEN** the human's open-melds list is empty
- **WHEN** the play screen renders
- **THEN** the rendered hand-row output equals the pre-change `renderHand()` output exactly (no extra whitespace, no marker characters, no second line)

#### Scenario: Wide meld block wraps to second line

- **GIVEN** the human has 4 ankans plus 13 concealed tiles (overflow case — the wide hand exceeds 80 columns)
- **WHEN** the play screen renders
- **THEN** the open-meld block appears on a SECOND row directly below the concealed-hand row, NOT truncated and NOT pushing the concealed row off-screen
