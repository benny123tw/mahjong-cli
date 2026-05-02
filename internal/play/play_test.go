package play

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestModelReflectsGameHandAndDiscards(t *testing.T) {
	g := game.New(7)
	m := NewWithGame(UnicodeRenderer{}, g)

	if got, want := len(m.Hand()), 13; got != want {
		t.Errorf("Model.Hand() length = %d, want %d", got, want)
	}
	for seat := range game.Seat(4) {
		if got := len(m.Pond(seat)); got != 0 {
			t.Errorf("Model.Pond(%d) at start = %d, want 0", seat, got)
		}
	}
}

func TestPeekKeyPopulatesShantenAndMachiCache(t *testing.T) {
	g := game.New(7)
	m := NewWithGame(UnicodeRenderer{}, g)

	if m.PeekShanten() != peekUnknown {
		t.Errorf(
			"Model.PeekShanten() before ?-press = %d, want sentinel %d",
			m.PeekShanten(),
			peekUnknown,
		)
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	mu, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	if mu.PeekShanten() == peekUnknown {
		t.Errorf("Model.PeekShanten() after ?-press = sentinel, want a real shanten value")
	}
}

func TestTsumoKeyOnYakulessHandReportsRejection(t *testing.T) {
	g := game.New(7)
	// Plant a winning shape with no yaku: open hand of all simples, no
	// tanyao-yakuhai-pinfu since hand is open. We just need a hand that
	// `calc.Analyze` returns nil for.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M5},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M6},
		{ID: tile.M7},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})
	g.SetTestOpen(game.SeatSouth, true)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 't'})
	mu, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	if !strings.Contains(mu.AckText(), "no yaku") && !strings.Contains(mu.AckText(), "cannot win") {
		t.Errorf(
			"Tsumo on yakuless hand AckText = %q, want a 'no yaku' rejection message",
			mu.AckText(),
		)
	}
}

func TestCallWindowFooterShowsOnlyLegalKeys(t *testing.T) {
	// Set up a state where the human (South) can pon East's discard.
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M8},
		{ID: tile.M9},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
	})
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.P5}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.RenderCallFooter()
	if !strings.Contains(footer, "[P]") {
		t.Errorf("Call footer = %q, want to contain [P] (pon is legal)", footer)
	}
	if !strings.Contains(footer, "Space") && !strings.Contains(footer, "Pass") {
		t.Errorf("Call footer = %q, want to contain Space/Pass", footer)
	}
}

func TestSpaceInCallWindowRecordsPass(t *testing.T) {
	g := game.New(7)
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.M5}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	mu, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	// Pass with no other claims advances to South's draw, which auto-fires
	// the draw and lands the model in AwaitingDiscard{South}.
	if _, ok := mu.GameState().(game.StateAwaitingDiscard); !ok {
		t.Errorf(
			"After Space pass with no claims, state = %T, want StateAwaitingDiscard (auto-drew on entering AwaitingDraw{South})",
			mu.GameState(),
		)
	}
}

// TestRenderHandInsertsGapBeforeDrawnTile confirms the Play Screen Layout
// drawn-tile separator: in AwaitingDiscard{Human} with 14 tiles, the rendered
// hand shows the leftmost 13 densely concatenated, then a one-tile-slot gap
// of horizontal whitespace, then the 14th tile.
func TestRenderHandInsertsGapBeforeDrawnTile(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		// 13 sorted tiles
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S1},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.Chun},
		// drawn 14th
		{ID: tile.M4},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	r := UnicodeRenderer{}
	m := NewWithGame(r, g)
	rendered := m.renderHand()

	// The drawn tile (M4) renders as `<glyph><vs15> ` — find the rightmost
	// occurrence and confirm there is at least Width() whitespace cells
	// immediately before it (the gap), in addition to the previous tile's
	// own trailing space.
	drawnGlyph := r.Tile(tile.Tile{ID: tile.M4})[0]
	idx := strings.LastIndex(rendered, drawnGlyph)
	if idx < 0 {
		t.Fatalf("rendered hand missing drawn tile glyph: rendered=%q", rendered)
	}
	prefix := rendered[:idx]
	trailing := 0
	for i := len(prefix) - 1; i >= 0; i-- {
		if prefix[i] == ' ' {
			trailing++
			continue
		}
		break
	}
	// Width() cells of inserted gap PLUS the previous tile's own trailing
	// space = at least Width() + 1 trailing whitespace before the drawn
	// tile's glyph.
	want := r.Width() + 1
	if trailing < want {
		t.Errorf(
			"AwaitingDiscard{Human}: expected ≥%d trailing whitespace before drawn tile, got %d. rendered=%q",
			want,
			trailing,
			rendered,
		)
	}
}

// TestRenderHandNoGapWhenNotAwaitingDiscard confirms the gap separator only
// appears in AwaitingDiscard{Human}: a 13-tile hand in AwaitingDraw or any
// other state SHALL render densely with no gap.
func TestRenderHandNoGapWhenNotAwaitingDiscard(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S1},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.Chun},
	})
	g.SetTestState(game.StateAwaitingDraw{Player: game.SeatSouth})

	r := UnicodeRenderer{}
	m := NewWithGame(r, g)
	rendered := m.renderHand()

	// Each tile rendered ends with a single trailing space (built into
	// UnicodeRenderer.Tile). A run of ≥ Width()+1 consecutive whitespace
	// would indicate a gap was inserted; that should NOT happen here.
	maxRun := 0
	cur := 0
	for i := 0; i < len(rendered); i++ {
		if rendered[i] == ' ' {
			cur++
			if cur > maxRun {
				maxRun = cur
			}
			continue
		}
		cur = 0
	}
	if maxRun > r.Width() {
		t.Errorf(
			"AwaitingDraw with 13-tile hand: found whitespace run of %d (>Width()=%d), expected dense rendering with no gap. rendered=%q",
			maxRun,
			r.Width(),
			rendered,
		)
	}
}

// TestCursorAtIndex13HighlightsDrawnTile confirms cursor handling still
// works after the drawn-tile gap renders: moving the cursor to index 13
// highlights the 14th (drawn) tile, not the gap.
func TestCursorAtIndex13HighlightsDrawnTile(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.S1},
		{ID: tile.S1},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.M4},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	r := UnicodeRenderer{}
	m := NewWithGame(r, g)
	// Move cursor to the rightmost (drawn) tile.
	for range 13 {
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'l'})
		m = updated.(Model)
	}
	if m.cursor != 13 {
		t.Fatalf("cursor after 13 right-moves = %d, want 13", m.cursor)
	}

	rendered := m.renderHand()
	// focusedTileStyle injects ANSI escape `\x1b[` before the highlighted
	// tile glyph. Since cursor=13, that escape should appear immediately
	// before the drawn-tile glyph (M4), not before any of the leftmost 13.
	drawnGlyph := r.Tile(tile.Tile{ID: tile.M4})[0]
	drawnIdx := strings.LastIndex(rendered, drawnGlyph)
	if drawnIdx < 0 {
		t.Fatalf("drawn tile glyph missing from rendered output: %q", rendered)
	}
	// Find the most recent ANSI start before drawnIdx.
	prefix := rendered[:drawnIdx]
	lastEsc := strings.LastIndex(prefix, "\x1b[")
	if lastEsc < 0 {
		t.Fatalf("no ANSI escape before drawn tile (expected cursor highlight): %q", rendered)
	}
	// Between lastEsc and drawnIdx there should be only the escape sequence
	// (ending in 'm') — no other tile glyphs. A simple check: no other
	// glyphs from the leftmost 13 should appear after lastEsc.
	tail := prefix[lastEsc:]
	for _, want := range []string{
		r.Tile(tile.Tile{ID: tile.Chun})[0],
		r.Tile(tile.Tile{ID: tile.S1})[0],
	} {
		if strings.Contains(tail, want) {
			t.Errorf(
				"cursor's ANSI escape appears too early — tile %q renders between escape and drawn tile, suggests cursor lands in gap. rendered=%q",
				want,
				rendered,
			)
		}
	}
}

// TestBotTickIsNotScheduledWhenHumanHasLegalPon confirms the call-window
// must wait for the human when they have a legal claim. If isBotTurn
// returned true unconditionally for StateAwaitingClaims, the 250ms tick
// would auto-pass before the player could press P or C.
func TestBotTickIsNotScheduledWhenHumanHasLegalPon(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M8},
		{ID: tile.M9},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
	})
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.P5}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	if cmd := m.Init(); cmd != nil {
		t.Errorf(
			"Init() with human-has-legal-pon scheduled a bot tick (cmd != nil); the human's call window must wait for input",
		)
	}
}

// TestBotTickIsScheduledWhenHumanHasNoLegalClaim confirms the auto-tick
// still fires for claim states the human can't act on, so bot auto-pass
// advances without manual input.
func TestBotTickIsScheduledWhenHumanHasNoLegalClaim(t *testing.T) {
	g := game.New(7)
	// Human's hand has no copy of the discard and no chi-tiles — nothing
	// to claim.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P1},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S1},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.Haku},
	})
	// East's discard is Chun; South can't pon (only one copy in hand? actually
	// zero copies) and can't chi (Chun is honor). Discarder is West (kamicha
	// of North, not South), so chi from West would be South's option only
	// if South is shimocha — South.Kamicha() == East, so chi-from-East is
	// the only legal chi for South. Use East as discarder of an honor tile.
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.Chun}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	if cmd := m.Init(); cmd == nil {
		t.Errorf(
			"Init() with no-legal-human-claim returned nil cmd; bot tick should be scheduled to auto-pass",
		)
	}
}

func TestBotTickAdvancesBotTurn(t *testing.T) {
	g := game.New(7)
	// State is already AwaitingDraw{East} at New; East is a bot.
	m := NewWithGame(UnicodeRenderer{}, g)

	// Init should schedule a bot tick when starting on a bot's turn.
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("Init() on bot's turn returned nil cmd, want tea.Tick")
	}

	// Drive the bot tick directly (skipping the wall-clock wait) and verify
	// the bot drew + discarded — state should be on the next player or
	// through to claims.
	updated, _ := m.Update(BotTickMsg{})
	mu, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	switch mu.GameState().(type) {
	case game.StateAwaitingDiscard, game.StateAwaitingClaims, game.StateAwaitingDraw:
		// All acceptable — bot may have drawn (next: discard), discarded
		// (next: claims), or claims may have resolved to next draw.
	default:
		t.Errorf(
			"After BotTickMsg on East's turn, state = %T, want one of AwaitingDiscard/AwaitingClaims/AwaitingDraw",
			mu.GameState(),
		)
	}
}

// fourteenTileTenpai14 is the 14-tile fixture used across riichi/ron tests.
// Discarding index 13 leaves a tenpai 13-tile shape with machi {4s, 7s}.
func fourteenTileTenpai14() []tile.Tile {
	return []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.Haku},
		{ID: tile.Haku},
		{ID: tile.M5}, // unrelated drawn tile to discard
	}
}

func TestRiichiKeyDeclaresRiichiWhenLegal(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, fourteenTileTenpai14())
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	m.cursor = 13
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	mu := updated.(Model)

	if _, ok := mu.GameState().(game.StateAwaitingClaims); !ok {
		t.Errorf(
			"after R-press in discard state with tenpai hand, state = %T, want StateAwaitingClaims",
			mu.GameState(),
		)
	}
	if mu.AckText() != "" {
		t.Errorf("ackText after legal riichi = %q, want empty", mu.AckText())
	}
}

func TestRiichiKeyRejectedWithDescriptiveAck(t *testing.T) {
	g := game.New(7)
	// 14-tile hand where discarding index 0 leaves a non-tenpai shape.
	noisy := []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M3},
		{ID: tile.M5},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.P1},
		{ID: tile.P3},
		{ID: tile.P5},
		{ID: tile.P7},
		{ID: tile.P9},
		{ID: tile.S1},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S7},
	}
	g.SetTestHand(game.SeatSouth, noisy)
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	m.cursor = 0
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	mu := updated.(Model)

	if !strings.Contains(mu.AckText(), "tenpai") {
		t.Errorf("ackText after illegal riichi = %q, want substring 'tenpai'", mu.AckText())
	}
}

// ronReadyHand is a 13-tile hand that wins yakufully on 7s (tanyao + pinfu shape).
func ronReadyHand() []tile.Tile {
	return []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.S5},
		{ID: tile.S6},
	}
}

func TestRonKeyDeclaresRonWhenLegal(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, ronReadyHand())
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	mu := updated.(Model)

	st, ok := mu.GameState().(game.StateRoundOver)
	if !ok {
		t.Fatalf(
			"after R-press in claims with ron-ready hand, state = %T, want StateRoundOver",
			mu.GameState(),
		)
	}
	if _, ok := st.Outcome.(game.OutcomeRon); !ok {
		t.Errorf("outcome = %T, want OutcomeRon", st.Outcome)
	}
}

func TestRonKeyRejectedWhenFuriten(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, ronReadyHand())
	// Plant 7s in own pond → permanent furiten.
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.S7}})
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	mu := updated.(Model)

	if _, ok := mu.GameState().(game.StateAwaitingClaims); !ok {
		t.Errorf(
			"after R-press with furiten, state = %T, want unchanged StateAwaitingClaims",
			mu.GameState(),
		)
	}
	if !strings.Contains(mu.AckText(), "furiten") {
		t.Errorf("ackText after furiten ron = %q, want substring 'furiten'", mu.AckText())
	}
}

func TestCallFooterShowsLiveRonWhenLegal(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, ronReadyHand())
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.RenderCallFooter()
	if !strings.Contains(footer, "[R]on") {
		t.Errorf("footer = %q, want [R]on label", footer)
	}
	if strings.Contains(footer, "(furiten)") {
		t.Errorf("footer = %q, should NOT contain (furiten) when not in furiten", footer)
	}
}

func TestCallFooterShowsFuritenSuffixWhenBlocked(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, ronReadyHand())
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.S7}})
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.RenderCallFooter()
	if !strings.Contains(footer, "(furiten)") {
		t.Errorf("footer = %q, want (furiten) suffix when permanent furiten blocks ron", footer)
	}
}

// fourteenTileWinningHand returns a 14-tile hand that wins yakufully on
// tsumo (tanyao + concealed): 234m 234p 234s 44m 5s6s7s.
func fourteenTileWinningHand() []tile.Tile {
	return []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M4},
		{ID: tile.M4},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
	}
}

func TestBotDispatchTsumoOnWinningHand(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatEast, fourteenTileWinningHand())
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatEast})

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	st, ok := mu.GameState().(game.StateRoundOver)
	if !ok {
		t.Fatalf(
			"after BotTickMsg with bot at winning hand, state = %T, want StateRoundOver",
			mu.GameState(),
		)
	}
	if _, ok := st.Outcome.(game.OutcomeTsumo); !ok {
		t.Errorf("outcome = %T, want OutcomeTsumo", st.Outcome)
	}
}

func TestBotDispatchRiichiOnTenpaiHand(t *testing.T) {
	g := game.New(7)
	// Plant tenpai-after-discard for SeatEast.
	tenpai := append([]tile.Tile(nil), tenpaiHand13Test()...)
	tenpai = append(tenpai, tile.Tile{ID: tile.M5})
	g.SetTestHand(game.SeatEast, tenpai)
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatEast})

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	cs, ok := mu.GameState().(game.StateAwaitingClaims)
	if !ok {
		t.Fatalf("state after bot riichi = %T, want StateAwaitingClaims", mu.GameState())
	}
	if cs.Discarder != game.SeatEast {
		t.Errorf("claims discarder = %d, want SeatEast", cs.Discarder)
	}
}

// tenpaiHand13Test mirrors riichi_test.go's tenpaiHandReady — duplicated here
// because internal/play can't see internal/game's package-private fixtures.
func tenpaiHand13Test() []tile.Tile {
	return []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.Haku},
		{ID: tile.Haku},
	}
}

func TestBotDispatchRonOnYakuBearingDiscard(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatNorth, ronReadyHand())
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	st, ok := mu.GameState().(game.StateRoundOver)
	if !ok {
		t.Fatalf("state after bot ron = %T, want StateRoundOver", mu.GameState())
	}
	out, ok := st.Outcome.(game.OutcomeRon)
	if !ok {
		t.Fatalf("outcome = %T, want OutcomeRon", st.Outcome)
	}
	if out.Winner != game.SeatNorth {
		t.Errorf("ron winner = %d, want SeatNorth", out.Winner)
	}
	if out.Loser != game.SeatEast {
		t.Errorf("ron loser = %d, want SeatEast", out.Loser)
	}
}

func TestBotDispatchPonOnYakuhaiDiscard(t *testing.T) {
	g := game.New(7)
	// Plant SeatNorth with two East-wind tiles for yakuhai pon.
	g.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.EastWind},
		{ID: tile.EastWind},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M5},
		{ID: tile.M6},
	})
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.EastWind}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	st, ok := mu.GameState().(game.StateAwaitingDiscard)
	if !ok {
		t.Fatalf("state after bot pon = %T, want StateAwaitingDiscard", mu.GameState())
	}
	if st.Player != game.SeatNorth {
		t.Errorf("post-pon active player = %d, want SeatNorth", st.Player)
	}
}

func TestBotDispatchAvoidsGenbutsuAgainstRiichiDeclarer(t *testing.T) {
	// The human (South) is in riichi with 5p in their pond. SeatNorth is
	// active and holds 5p (genbutsu) plus four isolated honors (East,
	// South, West, Haku) which would each score 1000 isolation — vastly
	// higher than the heavily-connected 5p (~99). Without danger awareness
	// the bot would discard a honor; with the K=2000 penalty on unknown
	// tiles the genbutsu wins.
	g := game.New(7)
	g.SetTestRiichiDeclared(game.SeatSouth, true)
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.P5}})
	g.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P5},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.Haku},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatNorth})

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	// After dispatch, the discard sits at the back of North's pond.
	northPond := mu.Pond(game.SeatNorth)
	if len(northPond) == 0 {
		t.Fatalf(
			"North's pond is empty after BotTickMsg; bot did not discard. State: %T",
			mu.GameState(),
		)
	}
	got := northPond[len(northPond)-1]
	if got.ID != tile.P5 {
		t.Errorf(
			"bot discarded %s, want 5p (the genbutsu). Without danger awareness an honor would have won on isolation; the 2000K penalty must dominate.",
			got,
		)
	}
}

func TestBotDispatchDoesNotSubmitForHuman(t *testing.T) {
	g := game.New(7)
	// Plant the human at a yaku-bearing winning shape on the East discard.
	// The bot dispatcher MUST NOT auto-submit a ClaimRon for the human —
	// only the human's keypress can declare their claim.
	g.SetTestHand(game.SeatSouth, ronReadyHand())
	g.SetTestState(
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.S7}, Discarder: game.SeatEast},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

	if st, ok := mu.GameState().(game.StateRoundOver); ok {
		if out, isRon := st.Outcome.(game.OutcomeRon); isRon && out.Winner == game.SeatSouth {
			t.Errorf(
				"bot dispatcher auto-submitted ron for the human (winner=South); the human's claim must come from their own keypress",
			)
		}
	}
}

func TestStatusBarReflectsMatchState(t *testing.T) {
	m := game.NewMatch(7)
	m.SetTestScore(game.SeatSouth, 27300)
	m.SetTestRiichiSticks(2)
	m.SetTestHonba(1)
	m.SetTestHandIndex(1) // East 2

	model := NewWithMatch(UnicodeRenderer{}, m)
	status := model.View().Content

	if !strings.Contains(status, "East 2") {
		t.Errorf("status bar does not contain 'East 2'. View:\n%s", status)
	}
	if !strings.Contains(status, "Honba 1") {
		t.Errorf("status bar does not contain 'Honba 1'. View:\n%s", status)
	}
	if !strings.Contains(status, "Riichi 2") {
		t.Errorf("status bar does not contain 'Riichi 2'. View:\n%s", status)
	}
	if !strings.Contains(status, "27300") {
		t.Errorf("status bar does not contain human score 27300. View:\n%s", status)
	}
}

func TestEndOfHandAckPanelOnRoundOver(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	// Force the current game into StateRoundOver with a non-dealer ron outcome.
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
		Winner: game.SeatSouth,
		Loser:  game.SeatEast,
		Tile:   tile.Tile{ID: tile.S7},
		Result: &calc.Result{Award: score.Award{Total: 1000, Base: 240}},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)

	if mu.pendingTransition == nil {
		t.Fatalf("pendingTransition is nil after RoundOver tick; expected ack to fire")
	}
	view := mu.View().Content
	if !strings.Contains(view, "ron") {
		t.Errorf("ack panel view does not mention 'ron'. View:\n%s", view)
	}
	if !strings.Contains(view, "South") || !strings.Contains(view, "East") {
		t.Errorf("ack panel view missing seat labels. View:\n%s", view)
	}
}

func TestKeypressOnAckAdvancesToNextHand(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
		Winner: game.SeatSouth,
		Loser:  game.SeatEast,
		Tile:   tile.Tile{ID: tile.S7},
		Result: &calc.Result{Award: score.Award{Total: 1000, Base: 240}},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	// First Update fires AdvanceFromOutcome and sets pendingTransition.
	model1, _ := model.Update(BotTickMsg{})
	mu1 := model1.(Model)
	if mu1.pendingTransition == nil {
		t.Fatalf("first tick did not set pendingTransition")
	}
	// Second Update with a keypress clears the pending transition.
	model2, _ := mu1.Update(tea.KeyPressMsg{Code: ' '})
	mu2 := model2.(Model)

	if mu2.pendingTransition != nil {
		t.Errorf("pendingTransition not cleared after keypress")
	}
	if mu2.GameState() == nil {
		t.Errorf("GameState() is nil after advance; expected the new hand's state")
	}
	// The match should now be at East 2, dealer = SeatSouth.
	if m.HandIndex() != 1 {
		t.Errorf("after non-renchan ron, HandIndex = %d, want 1", m.HandIndex())
	}
	if m.Dealer() != game.SeatSouth {
		t.Errorf("after rotation, Dealer = %d, want SeatSouth", m.Dealer())
	}
}

func TestStandingsScreenOnHanchanCompletion(t *testing.T) {
	m := game.NewMatch(7)
	m.SetTestHandIndex(7) // South 4
	m.SetTestScore(game.SeatEast, 26500)
	m.SetTestScore(game.SeatSouth, 24500)
	m.SetTestScore(game.SeatWest, 27500)
	m.SetTestScore(game.SeatNorth, 21500)
	dealer := m.Dealer()
	nonDealer := dealer.Next()
	o := game.OutcomeRon{
		Winner: nonDealer,
		Loser:  dealer,
		Tile:   tile.Tile{ID: tile.S7},
		Result: &calc.Result{Award: score.Award{Total: 1000, Base: 240}},
	}
	if _, err := m.AdvanceFromOutcome(o); err != nil {
		t.Fatalf("AdvanceFromOutcome err: %v", err)
	}
	if !m.IsFinished() {
		t.Fatalf("precondition: match should be finished at South 4 with non-renchan outcome")
	}

	model := NewWithMatch(UnicodeRenderer{}, m)
	view := model.View().Content

	if !strings.Contains(view, "Hanchan complete") {
		t.Errorf("standings view missing 'Hanchan complete' header. View:\n%s", view)
	}
	for _, label := range []string{"East", "South", "West", "North"} {
		if !strings.Contains(view, label) {
			t.Errorf("standings view missing seat %q. View:\n%s", label, view)
		}
	}
	if !strings.Contains(view, "hanchan-complete") {
		t.Errorf("standings view missing reason 'hanchan-complete'. View:\n%s", view)
	}

	// Quit key returns tea.Quit.
	_, cmd := model.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Errorf("standings 'q' returned nil cmd, want tea.Quit")
	}
}
