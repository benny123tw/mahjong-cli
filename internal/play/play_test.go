package play

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
	"github.com/benny123tw/mahjong-cli/internal/riichi/yaku"
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

// TestAutoDrawHumanJumpsCursorToDrawnTile confirms the cursor lands on
// the drawn tile (last index, after the visual gap) when the human auto-
// draws. This is the standard riichi UX: tsumogiri (discard the drawn
// tile) is the most common play, so the cursor should already be there.
func TestAutoDrawHumanJumpsCursorToDrawnTile(t *testing.T) {
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

	m := NewWithGame(UnicodeRenderer{}, g)
	m.cursor = 5
	m = m.autoDrawHuman()

	if got := len(m.game.Hand(game.SeatSouth)); got != 14 {
		t.Fatalf("hand length after autoDrawHuman = %d, want 14", got)
	}
	if m.cursor != 13 {
		t.Errorf("cursor after autoDrawHuman = %d, want 13 (drawn tile)", m.cursor)
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

// TestNoCallWindowOnHumansOwnDiscard verifies the human can't pon/chi/kan/ron
// their own just-discarded tile. Even when CanPon would return true (the
// human still has 2 matching tiles after discarding the 3rd), the call
// window MUST NOT prompt because the discarder cannot claim their own
// discard. Real-game bug: discarding from a triplet-in-hand opens a
// nonsensical pon offer.
func TestNoCallWindowOnHumansOwnDiscard(t *testing.T) {
	g := game.New(7)
	// Human keeps 2 P5s in hand after notionally discarding a 3rd P5 —
	// CanPon would return true on this hand against discard P5.
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
		game.StateAwaitingClaims{Discard: tile.Tile{ID: tile.P5}, Discarder: game.SeatSouth},
	)

	m := NewWithGame(UnicodeRenderer{}, g)
	if cmd := m.Init(); cmd == nil {
		t.Errorf(
			"Init() with human-as-discarder did NOT schedule a bot tick; " +
				"call window should be skipped (you can't claim your own discard)",
		)
	}
	if cf := m.RenderCallFooter(); cf != "" {
		t.Errorf(
			"RenderCallFooter() returned %q on human's own discard; "+
				"expected empty (no claim prompt). View:\n%s",
			cf, cf,
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
	if !strings.Contains(view, "RON") {
		t.Errorf("ack panel view does not mention RON. View:\n%s", view)
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

func TestQuestionMarkTogglesPeekVisibility(t *testing.T) {
	g := game.New(7)
	m := NewWithGame(UnicodeRenderer{}, g)

	if m.peekVisible {
		t.Fatalf("initial peekVisible = true, want false")
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	mu := updated.(Model)
	if !mu.peekVisible {
		t.Errorf("after first ?-press, peekVisible = false, want true")
	}
	updated2, _ := mu.Update(tea.KeyPressMsg{Code: '?'})
	mu2 := updated2.(Model)
	if mu2.peekVisible {
		t.Errorf("after second ?-press, peekVisible = true, want false")
	}
}

func TestRenderFooterShowsWaitLineWhenPeekVisibleAndTenpai(t *testing.T) {
	g := game.New(7)
	// Hand 1m2m3m4p5p6p7s8s9s2z2z3z3z is tenpai waiting on the second 4p/7p
	// (… actually it's a slightly different shape — let me use a known
	// tenpai). Use 1m2m3m4p5p6p7s8s9s1m1m2z2z which has been used elsewhere
	// in this file as a tenpai fixture (waits on 1m).
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	m = m.handlePeek() // populate cache + flip peekVisible to true
	if !m.peekVisible {
		t.Fatalf("peekVisible after handlePeek = false, want true")
	}
	footer := m.renderFooter()
	if !strings.Contains(footer, "Wait:") {
		t.Errorf("footer missing 'Wait:' prefix when peek visible + tenpai. Footer:\n%s", footer)
	}
}

func TestRenderFooterShowsNotTenpaiWhenPeekVisibleAndNotTenpai(t *testing.T) {
	g := game.New(7)
	// Plant a far-from-tenpai hand: 13 random tiles with no shape.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M3},
		{ID: tile.M5},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.P2},
		{ID: tile.P4},
		{ID: tile.P6},
		{ID: tile.S1},
		{ID: tile.S3},
		{ID: tile.S9},
		{ID: tile.EastWind},
		{ID: tile.Haku},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	m = m.handlePeek()
	if !m.peekVisible {
		t.Fatalf("peekVisible after handlePeek = false, want true")
	}
	footer := m.renderFooter()
	if !strings.Contains(footer, "Wait: (not tenpai)") {
		t.Errorf("footer missing 'Wait: (not tenpai)' for non-tenpai hand. Footer:\n%s", footer)
	}
}

func TestFuritenBadgeAppearsWhenHumanTenpaiAndFuriten(t *testing.T) {
	g := game.New(7)
	// Tenpai hand waiting on 1m.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	// Plant a 1m in the human's own pond → permanent furiten.
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.M1}})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.renderFooter()
	if !strings.Contains(footer, "FURITEN") {
		t.Errorf(
			"Unicode footer missing [FURITEN] badge for tenpai+furiten human. Footer:\n%s",
			footer,
		)
	}
}

func TestFuritenBadgeASCIIUsesParenForm(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.M1}})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(ASCIIRenderer{}, g)
	footer := m.renderFooter()
	if !strings.Contains(footer, "(furiten)") {
		t.Errorf("ASCII footer missing (furiten) form. Footer:\n%s", footer)
	}
	if strings.Contains(footer, "[FURITEN]") {
		t.Errorf("ASCII footer should not contain [FURITEN]. Footer:\n%s", footer)
	}
}

func TestFuritenBadgeAbsentWhenNotTenpai(t *testing.T) {
	g := game.New(7)
	// Far-from-tenpai hand.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M3},
		{ID: tile.M5},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.P2},
		{ID: tile.P4},
		{ID: tile.P6},
		{ID: tile.S1},
		{ID: tile.S3},
		{ID: tile.S9},
		{ID: tile.EastWind},
		{ID: tile.Haku},
	})
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.M1}})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.renderFooter()
	if strings.Contains(footer, "FURITEN") || strings.Contains(footer, "(furiten)") {
		t.Errorf("non-tenpai hand should NOT show furiten badge. Footer:\n%s", footer)
	}
}

func TestFuritenBadgeAbsentInCallWindow(t *testing.T) {
	g := game.New(7)
	// Tenpai+furiten human.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.M1}})
	g.SetTestState(game.StateAwaitingClaims{
		Discard:   tile.Tile{ID: tile.M1},
		Discarder: game.SeatEast,
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	footer := m.renderFooter()
	// Call-window footer renders RenderCallFooter which uses the inline
	// [R]on (furiten) suffix. We assert the standalone [FURITEN] badge is NOT
	// emitted (the `(furiten)` substring may appear via Ron-button suffix —
	// only check for the standalone uppercase badge form).
	if strings.Contains(footer, "[FURITEN]") {
		t.Errorf("call window should not emit standalone [FURITEN] badge. Footer:\n%s", footer)
	}
}

func TestDispatchBotDiscardUsesFoldModeAtHighShanten(t *testing.T) {
	// Bot at shanten >= 2 (deliberately broken: 14 disconnected tiles, all
	// honors + isolated suit singletons). Human in riichi with 5p in pond.
	// Bot's hand contains 5p (genbutsu). Fold mode SHALL route to
	// FoldDiscard which picks the genbutsu over higher-isolation unknowns.
	g := game.New(7)
	g.SetTestRiichiDeclared(game.SeatSouth, true)
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.P5}})
	g.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M3},
		{ID: tile.M5},
		{ID: tile.M7},
		{ID: tile.M9},
		{ID: tile.P5}, // genbutsu
		{ID: tile.S1},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S7},
		{ID: tile.S9},
		{ID: tile.EastWind},
		{ID: tile.WestWind},
		{ID: tile.Haku},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatNorth})

	// Sanity: confirm the planted hand is at shanten >= 2 so fold-mode
	// routing actually fires (the dispatcher gates on shanten >= 2).
	if got := hand.Shanten(hand.Hand{Concealed: g.Hand(game.SeatNorth)}); got < 2 {
		t.Fatalf("planted hand shanten = %d, want >= 2 for fold-mode test", got)
	}

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)

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
			"fold-mode bot discarded %s, want 5p (the genbutsu). Fold mode at shanten>=2 with riichi declared MUST pick the safest tile.",
			got,
		)
	}
}

func TestDispatchBotDiscardUsesPushModeAtLowShanten(t *testing.T) {
	// Bot at shanten = 0 (tenpai). Even with riichi declared and a genbutsu
	// available, the dispatcher SHALL route to DangerAwarePickDiscard (push
	// mode) — fold-mode gates on shanten >= 2. The behavioral assertion is
	// that the discard equals what DangerAwarePickDiscard would pick on the
	// same fixture. This proves the dispatcher routes correctly without
	// requiring push and fold to differ on this hand.
	g := game.New(7)
	g.SetTestRiichiDeclared(game.SeatSouth, true)
	g.SetTestPond(game.SeatSouth, []tile.Tile{{ID: tile.P5}})
	// Tenpai shape: 1m2m3m 4p5p6p 7s8s9s 5p East East + drawn extra. The 5p
	// in hand is genbutsu; East is unknown-danger isolated honor.
	g.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.P5},
		{ID: tile.EastWind},
		{ID: tile.EastWind},
		{ID: tile.NorthWind},
		{ID: tile.NorthWind},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatNorth})

	if got := hand.Shanten(hand.Hand{Concealed: g.Hand(game.SeatNorth)}); got > 1 {
		t.Fatalf("planted hand shanten = %d, want <= 1 for push-mode test", got)
	}

	m := NewWithGame(UnicodeRenderer{}, g)
	// Compute what DangerAwarePickDiscard would pick on this fixture.
	bot := game.Bot{Seat: game.SeatNorth, Rng: g.Wall().Rand()}
	hd := g.Hand(game.SeatNorth)
	danger := m.assembleDangerMap(game.SeatNorth, hd)
	wantIdx := bot.DangerAwarePickDiscard(hd, danger)
	wantTile := hd[wantIdx]

	updated, _ := m.Update(BotTickMsg{})
	mu := updated.(Model)
	northPond := mu.Pond(game.SeatNorth)
	if len(northPond) == 0 {
		t.Fatalf("North's pond is empty after BotTickMsg")
	}
	got := northPond[len(northPond)-1]
	if got.ID != wantTile.ID {
		t.Errorf(
			"push-mode bot discarded %s, want %s (DangerAwarePickDiscard's pick on the same fixture)",
			got,
			wantTile,
		)
	}
}

func TestRenderHandIncludesOpenMeldsWithSeatMarkers(t *testing.T) {
	g := game.New(7)
	// 13-tile concealed hand (any sensible shape).
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  game.SeatEast,
	})
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:  game.MeldChi,
		Tiles: []tile.Tile{{ID: tile.M2}, {ID: tile.M3}, {ID: tile.M4}},
		From:  game.SeatWest,
	})
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:    game.MeldKan,
		KanKind: game.KanAnkan,
		Tiles: []tile.Tile{
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
		},
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	out := m.renderHand()

	if !strings.Contains(out, "[E]") {
		t.Errorf("renderHand output missing [E] marker for pon-from-East. Output:\n%s", out)
	}
	if !strings.Contains(out, "[W]") {
		t.Errorf("renderHand output missing [W] marker for chi-from-West. Output:\n%s", out)
	}
	if !strings.Contains(out, "[A]") {
		t.Errorf("renderHand output missing [A] marker for ankan. Output:\n%s", out)
	}
}

func TestRenderHandPreservesByteIdenticalOutputWhenNoMelds(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	out := m.renderHand()

	for _, marker := range []string{"[E]", "[S]", "[W]", "[N]", "[A]"} {
		if strings.Contains(out, marker) {
			t.Errorf(
				"renderHand output contains %q with no melds present. Output:\n%s",
				marker,
				out,
			)
		}
	}
}

func TestRenderOpenMeldsMarkerPointsAtCalledTile(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	// Chi: tiles = [c1, c2, called]. Called tile is the 3rd (M4), marker
	// should sit between M3 and M4.
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:  game.MeldChi,
		Tiles: []tile.Tile{{ID: tile.M2}, {ID: tile.M3}, {ID: tile.M4}},
		From:  game.SeatWest,
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	out := m.renderOpenMeldsForSeat(game.SeatSouth)

	r := UnicodeRenderer{}
	m2Glyph := strings.Join(r.Tile(tile.Tile{ID: tile.M2}), "\n")
	m3Glyph := strings.Join(r.Tile(tile.Tile{ID: tile.M3}), "\n")
	m4Glyph := strings.Join(r.Tile(tile.Tile{ID: tile.M4}), "\n")

	idxM2 := strings.Index(out, m2Glyph)
	idxM3 := strings.Index(out, m3Glyph)
	idxM4 := strings.Index(out, m4Glyph)
	idxMarker := strings.Index(out, "[W]")
	if idxM2 < 0 || idxM3 < 0 || idxM4 < 0 || idxMarker < 0 {
		t.Fatalf("expected all of [W], M2, M3, M4 in output. Output:\n%s", out)
	}
	if !(idxM2 < idxM3 && idxM3 < idxMarker && idxMarker < idxM4) {
		t.Errorf(
			"expected order: M2 < M3 < [W] < M4 (marker pointing at called tile M4). "+
				"Got M2=%d M3=%d [W]=%d M4=%d. Output:\n%s",
			idxM2, idxM3, idxMarker, idxM4, out,
		)
	}
}

func TestRenderOpenMeldsAnkanKeepsMarkerAsPrefix(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:    game.MeldKan,
		KanKind: game.KanAnkan,
		Tiles: []tile.Tile{
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
		},
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	out := m.renderOpenMeldsForSeat(game.SeatSouth)

	r := UnicodeRenderer{}
	eastGlyph := strings.Join(r.Tile(tile.Tile{ID: tile.EastWind}), "\n")

	idxMarker := strings.Index(out, "[A]")
	idxFirstEast := strings.Index(out, eastGlyph)
	if idxMarker < 0 || idxFirstEast < 0 {
		t.Fatalf("expected [A] and East tile in output. Output:\n%s", out)
	}
	if idxMarker >= idxFirstEast {
		t.Errorf(
			"[A] should precede the first ankan tile (no called tile to point at). "+
				"Got [A]=%d, first East=%d. Output:\n%s",
			idxMarker, idxFirstEast, out,
		)
	}
}

// TestEndPanelRonRevealsHandsAndYakuBreakdown verifies that on a ron
// outcome the end-of-hand panel reveals all four seats' hands plus the
// yaku, totals, and deltas — per the End-of-Hand Acknowledgement
// requirement (the enriched reveal panel).
func TestEndPanelRonRevealsHandsAndYakuBreakdown(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()

	// Plant a known concealed hand on each seat; South contains 5p (winning tile).
	cur.SetTestHand(game.SeatEast, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
	})
	cur.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M9},
		{ID: tile.M9},
	})
	cur.SetTestHand(game.SeatWest, []tile.Tile{
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S4},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.Haku},
		{ID: tile.Haku},
		{ID: tile.Haku},
		{ID: tile.Chun},
	})
	cur.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M7},
		{ID: tile.M8},
		{ID: tile.M9},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.P9},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.Hatsu},
		{ID: tile.Hatsu},
		{ID: tile.NorthWind},
		{ID: tile.NorthWind},
	})

	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
		Winner: game.SeatSouth,
		Loser:  game.SeatEast,
		Tile:   tile.Tile{ID: tile.P5},
		Result: &calc.Result{
			YakuMatches: []yaku.Match{
				{Name: "Riichi", Han: 1},
				{Name: "Pinfu", Han: 1},
				{Name: "Tanyao", Han: 1},
			},
			Han:   3,
			Fu:    30,
			Award: score.Award{Han: 3, Fu: 30, Base: 480, Total: 3900},
		},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)
	view := mu.View().Content

	// Header: kind, winner, discarder.
	if !strings.Contains(view, "RON") {
		t.Errorf("end panel missing RON header. View:\n%s", view)
	}
	if !strings.Contains(view, "South") || !strings.Contains(view, "East") {
		t.Errorf("end panel missing winner/discarder seat names. View:\n%s", view)
	}

	// Yaku breakdown.
	if !strings.Contains(view, "Riichi 1") ||
		!strings.Contains(view, "Pinfu 1") ||
		!strings.Contains(view, "Tanyao 1") {
		t.Errorf("end panel missing per-yaku name+han entries. View:\n%s", view)
	}

	// Totals line.
	if !strings.Contains(view, "Han 3") ||
		!strings.Contains(view, "Fu 30") ||
		!strings.Contains(view, "Base 480") {
		t.Errorf("end panel missing Han/Fu/Base totals. View:\n%s", view)
	}

	// Deltas (East loses 3900, South gains 3900).
	if !strings.Contains(view, "-3900") || !strings.Contains(view, "+3900") {
		t.Errorf("end panel missing per-seat deltas. View:\n%s", view)
	}

	// Footer.
	if !strings.Contains(view, "[Any key — Continue]") {
		t.Errorf("end panel missing footer terminator. View:\n%s", view)
	}

	// Every seat's concealed hand should be face-up. Probe one tile per seat
	// (using glyph from the test renderer).
	r := UnicodeRenderer{}
	probes := []struct {
		seat string
		tile tile.Tile
	}{
		{"East", tile.Tile{ID: tile.S7}},
		{"West", tile.Tile{ID: tile.Chun}},
		{"North", tile.Tile{ID: tile.NorthWind}},
	}
	for _, p := range probes {
		glyph := strings.Join(r.Tile(p.tile), "\n")
		if !strings.Contains(view, glyph) {
			t.Errorf("end panel does not reveal %s's hand (missing glyph %q). View:\n%s",
				p.seat, glyph, view)
		}
	}
}

// TestEndPanelRonWinningTileIsHighlighted asserts the winning tile in
// the winner's hand row is bold (ANSI \x1b[1m) and that the winner's row
// begins with "[W] ". Per the End-of-Hand Acknowledgement requirement.
func TestEndPanelRonWinningTileIsHighlighted(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	// Plant non-P5 hands on East/West/North so the first P5 glyph in the
	// rendered view is the winner's (highlighted) tile in South's row.
	mTiles := []tile.Tile{
		{ID: tile.M1},
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
		{ID: tile.S4},
	}
	cur.SetTestHand(game.SeatEast, mTiles)
	cur.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M9},
		{ID: tile.M9},
	})
	cur.SetTestHand(game.SeatWest, []tile.Tile{
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.Haku},
		{ID: tile.Hatsu},
		{ID: tile.Chun},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.NorthWind},
		{ID: tile.M1},
	})
	cur.SetTestHand(game.SeatNorth, mTiles)
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
		Winner: game.SeatSouth,
		Loser:  game.SeatEast,
		Tile:   tile.Tile{ID: tile.P5},
		Result: &calc.Result{
			YakuMatches: []yaku.Match{{Name: "Riichi", Han: 1}},
			Han:         1,
			Fu:          30,
			Award:       score.Award{Han: 1, Fu: 30, Base: 480, Total: 1500},
		},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)
	view := mu.View().Content

	// focusedTileStyle uses Bold(true) + Foreground(212) which lipgloss
	// emits as the SGR sequence "\x1b[1;38;5;212m" — bold + 256-color fg.
	// The presence of this exact prefix immediately before the winning
	// glyph in the winner's row is the highlight signal.
	const focusedSGR = "\x1b[1;38;5;212m"
	r := UnicodeRenderer{}
	winGlyph := strings.Join(r.Tile(tile.Tile{ID: tile.P5}), "\n")

	southIdx := strings.Index(view, "South")
	if southIdx < 0 {
		t.Fatalf("end panel does not mention South. View:\n%s", view)
	}
	tail := view[southIdx:]
	beforeWin, _, found := strings.Cut(tail, winGlyph)
	if !found {
		t.Fatalf("winning glyph not present in South's row. View:\n%s", view)
	}
	if !strings.Contains(beforeWin, focusedSGR) {
		t.Errorf(
			"no focused-tile SGR before the winning %s glyph; expected highlight. "+
				"beforeWin=%q",
			winGlyph, beforeWin,
		)
	}

	// Winner row begins with "[W] " (after stripping leading style escapes).
	if !strings.Contains(view, "[W] ") {
		t.Errorf("winner-row prefix [W] missing. View:\n%s", view)
	}
}

// TestEndPanelFooterReplacesActionFooter asserts the panel-active footer
// is "[Any key — Continue]" and that the normal action-footer keys
// ("Move", "Discard", "Riichi", etc.) are absent. Per the End-of-Hand
// Acknowledgement requirement.
func TestEndPanelFooterReplacesActionFooter(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRyuukyoku{
		TenpaiPlayers: []game.Seat{game.SeatSouth},
	}})
	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)
	view := mu.View().Content

	if !strings.Contains(view, "[Any key — Continue]") {
		t.Errorf("end panel missing terminator footer. View:\n%s", view)
	}
	for _, banned := range []string{"Move", "Discard", "Riichi", "Tsumo"} {
		if strings.Contains(view, banned) {
			t.Errorf(
				"end panel leaks action-footer label %q "+
					"(action footer should be replaced). View:\n%s",
				banned, view,
			)
		}
	}
}

// TestEndPanelRyuukyokuLabelsAndPayments parameterized over all 5 cases
// from the ryuukyoku-payment example table (0/4, 1/3, 2/2, 3/1, 4/0).
// Per the End-of-Hand Acknowledgement requirement.
func TestEndPanelRyuukyokuLabelsAndPayments(t *testing.T) {
	cases := []struct {
		name         string
		tenpai       []game.Seat
		wantDeltas   string
		expectations [4]string
	}{
		{
			name:   "0 tenpai (0/4)",
			tenpai: []game.Seat{},
			// Per spec: 0 tenpai → all deltas 0.
			expectations: [4]string{"noten", "noten", "noten", "noten"},
		},
		{
			name:         "1 tenpai (1/3) — South",
			tenpai:       []game.Seat{game.SeatSouth},
			wantDeltas:   "+3000",
			expectations: [4]string{"noten", "tenpai", "noten", "noten"},
		},
		{
			name:         "2 tenpai (2/2) — South+West",
			tenpai:       []game.Seat{game.SeatSouth, game.SeatWest},
			wantDeltas:   "+1500",
			expectations: [4]string{"noten", "tenpai", "tenpai", "noten"},
		},
		{
			name:         "3 tenpai (3/1) — North noten",
			tenpai:       []game.Seat{game.SeatEast, game.SeatSouth, game.SeatWest},
			wantDeltas:   "-3000",
			expectations: [4]string{"tenpai", "tenpai", "tenpai", "noten"},
		},
		{
			name:   "4 tenpai (4/0)",
			tenpai: []game.Seat{game.SeatEast, game.SeatSouth, game.SeatWest, game.SeatNorth},
			// Per spec: 4 tenpai → all deltas 0.
			expectations: [4]string{"tenpai", "tenpai", "tenpai", "tenpai"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := game.NewMatch(7)
			cur := m.CurrentGame()
			cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRyuukyoku{
				TenpaiPlayers: c.tenpai,
			}})

			model := NewWithMatch(UnicodeRenderer{}, m)
			updated, _ := model.Update(BotTickMsg{})
			mu := updated.(Model)
			view := mu.View().Content

			if !strings.Contains(view, "RYUUKYOKU") {
				t.Errorf("missing RYUUKYOKU header. View:\n%s", view)
			}
			// Inline tenpai/noten label per seat row.
			seats := []string{"East", "South", "West", "North"}
			for i, seat := range seats {
				wantTag := c.expectations[i]
				seatIdx := strings.Index(view, seat)
				if seatIdx < 0 {
					t.Errorf("seat %s not in view. View:\n%s", seat, view)
					continue
				}
				// The row's tag appears after the seat label and before the
				// next seat label or the trailing deltas line.
				rowEnd := len(view)
				if i+1 < len(seats) {
					if nextIdx := strings.Index(view[seatIdx:], seats[i+1]); nextIdx >= 0 {
						rowEnd = seatIdx + nextIdx
					}
				} else if dIdx := strings.Index(view[seatIdx:], "(→"); dIdx >= 0 {
					// Deltas row contains "(→<total>)" — stop before it.
					rowEnd = seatIdx + dIdx
				}
				rowText := view[seatIdx:rowEnd]
				if !strings.Contains(rowText, wantTag) {
					t.Errorf(
						"seat %s row missing tag %q. row=%q",
						seat, wantTag, rowText,
					)
				}
			}
			// Deltas row signature for non-zero cases.
			if c.wantDeltas != "" && !strings.Contains(view, c.wantDeltas) {
				t.Errorf(
					"missing expected deltas substring %q. View:\n%s",
					c.wantDeltas, view,
				)
			}
			if !strings.Contains(view, "[Any key — Continue]") {
				t.Errorf("missing footer terminator. View:\n%s", view)
			}
		})
	}
}

// TestEndPanelTsumoRevealsHandsAndYakuBreakdown asserts the tsumo
// outcome surfaces the same reveal + yaku/han/fu/deltas content as ron,
// with a TSUMO header (no "from <seat>" clause). Per the End-of-Hand
// Acknowledgement requirement.
func TestEndPanelTsumoRevealsHandsAndYakuBreakdown(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	cur.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M9},
		{ID: tile.M9},
	})
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeTsumo{
		Winner: game.SeatSouth,
		Tile:   tile.Tile{ID: tile.P5},
		Result: &calc.Result{
			YakuMatches: []yaku.Match{
				{Name: "Riichi", Han: 1},
				{Name: "Pinfu", Han: 1},
				{Name: "Tanyao", Han: 1},
			},
			Han:   3,
			Fu:    30,
			Award: score.Award{Han: 3, Fu: 30, Base: 480, Total: 4000},
		},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)
	view := mu.View().Content

	if !strings.Contains(view, "TSUMO") {
		t.Errorf("end panel missing TSUMO header. View:\n%s", view)
	}
	// Tsumo header SHALL NOT include a "from <seat>" clause.
	if strings.Contains(view, "TSUMO — South wins from") {
		t.Errorf("TSUMO header should not contain 'from <seat>'. View:\n%s", view)
	}
	if !strings.Contains(view, "South") {
		t.Errorf("end panel missing winner seat name. View:\n%s", view)
	}
	if !strings.Contains(view, "Riichi 1") || !strings.Contains(view, "Pinfu 1") {
		t.Errorf("end panel missing yaku entries. View:\n%s", view)
	}
	if !strings.Contains(view, "Han 3") ||
		!strings.Contains(view, "Fu 30") ||
		!strings.Contains(view, "Base 480") {
		t.Errorf("end panel missing totals line. View:\n%s", view)
	}
	if !strings.Contains(view, "[Any key — Continue]") {
		t.Errorf("end panel missing footer terminator. View:\n%s", view)
	}
}

// TestEndPanelRonChankanLabelsHeader verifies that when the winning
// hand's yaku list contains "Chankan" (the engine's signal for a
// chankan-ron), the panel header reads "CHANKAN RON" instead of "RON".
// Per the End-of-Hand Acknowledgement requirement.
func TestEndPanelRonChankanLabelsHeader(t *testing.T) {
	m := game.NewMatch(7)
	cur := m.CurrentGame()
	cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
		Winner: game.SeatSouth,
		Loser:  game.SeatEast,
		Tile:   tile.Tile{ID: tile.M5},
		Result: &calc.Result{
			YakuMatches: []yaku.Match{
				{Name: "Riichi", Han: 1},
				{Name: "Chankan", Han: 1},
			},
			Han:   2,
			Fu:    30,
			Award: score.Award{Han: 2, Fu: 30, Base: 480, Total: 2000},
		},
	}})

	model := NewWithMatch(UnicodeRenderer{}, m)
	updated, _ := model.Update(BotTickMsg{})
	mu := updated.(Model)
	view := mu.View().Content

	if !strings.Contains(view, "CHANKAN RON") {
		t.Errorf(
			"end panel should label chankan-ron in the header. View:\n%s",
			view,
		)
	}
	// And NOT plain RON header alongside (the kind word is exclusive).
	// We check that "RON" appears only as part of "CHANKAN RON" — i.e.,
	// the only occurrence is preceded by "CHANKAN ".
	idx := strings.Index(view, "RON")
	if idx > 0 && !strings.HasSuffix(view[:idx], "CHANKAN ") {
		t.Errorf("plain RON appears outside CHANKAN RON. View:\n%s", view)
	}
}

// TestOpponentZonesUnchangedWithZeroMelds asserts that when no opponent
// has any open melds, the per-zone rendering is structurally identical
// to the pre-change output: label, optional pond, no extra blank line
// where the meld block would have gone. Per the Play Screen Layout
// zero-meld scenario.
func TestOpponentZonesUnchangedWithZeroMelds(t *testing.T) {
	m := game.NewMatch(7)
	model := NewWithMatch(UnicodeRenderer{}, m)

	// With zero melds on each opponent, the zone output must NOT contain
	// a doubled-newline pattern between label and pond/back-row that
	// would only appear if the meld renderer added an empty line.
	zones := []struct {
		name string
		out  string
	}{
		{"Kamicha", model.renderKamichaColumn()},
		{"Shimocha", model.renderShimochaColumn()},
		{"Toimen", model.renderToimenRow()},
	}
	for _, z := range zones {
		if strings.Contains(z.out, "\n\n") {
			t.Errorf(
				"%s zone with zero melds contains \\n\\n; suggests meld-block "+
					"insertion is leaking an empty line. Output:\n%s",
				z.name, z.out,
			)
		}
	}
}

// TestOpponentZonesIncludeMelds verifies that each opponent's zone in
// the four-quadrant layout shows that seat's open melds between the
// seat label and the seat's other content (back-row for Toimen,
// discard pond for Kamicha and Shimocha). Per the Play Screen Layout
// requirement (opponent-zone rendering).
func TestOpponentZonesIncludeMelds(t *testing.T) {
	cases := []struct {
		name       string
		seat       game.Seat
		seatLabel  string
		meldTileID uint8
		fromSeat   game.Seat
		fromMarker string
	}{
		{
			"Kamicha · East", game.SeatEast, "Kamicha · East",
			tile.P5, game.SeatSouth, "[S]",
		},
		{
			"Toimen — North", game.SeatNorth, "Toimen — North",
			tile.M1, game.SeatWest, "[W]",
		},
		{
			"Shimocha · West", game.SeatWest, "Shimocha · West",
			tile.P9, game.SeatEast, "[E]",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := game.NewMatch(7)
			cur := m.CurrentGame()
			cur.SetTestMeld(c.seat, game.Meld{
				Kind: game.MeldPon,
				Tiles: []tile.Tile{
					{ID: c.meldTileID},
					{ID: c.meldTileID},
					{ID: c.meldTileID},
				},
				From: c.fromSeat,
			})
			model := NewWithMatch(UnicodeRenderer{}, m)
			view := model.View().Content

			r := UnicodeRenderer{}
			meldGlyph := strings.Join(r.Tile(tile.Tile{ID: c.meldTileID}), "\n")

			labelIdx := strings.Index(view, c.seatLabel)
			if labelIdx < 0 {
				t.Fatalf("seat label %q missing from view. View:\n%s", c.seatLabel, view)
			}
			meldIdx := strings.Index(view[labelIdx:], meldGlyph)
			if meldIdx < 0 {
				t.Fatalf("meld glyph not found AFTER seat label. View:\n%s", view)
			}
			markerIdx := strings.Index(view[labelIdx:], c.fromMarker)
			if markerIdx < 0 {
				t.Fatalf("seat-source marker %q missing AFTER label. View:\n%s",
					c.fromMarker, view)
			}
		})
	}
}

// TestRenderOpponentMeldsZeroMeldCase asserts that a seat with no open
// melds produces an empty string, so the four-quadrant layout stays
// byte-identical to the pre-change rendering. Per the Play Screen Layout
// requirement (zero-meld scenario).
func TestRenderOpponentMeldsZeroMeldCase(t *testing.T) {
	g := game.New(7)
	m := NewWithGame(UnicodeRenderer{}, g)

	for _, seat := range []game.Seat{game.SeatEast, game.SeatNorth, game.SeatWest} {
		if got := m.renderOpponentMelds(seat, kamichaZoneWidth); got != "" {
			t.Errorf("renderOpponentMelds(%v, %d) with no melds = %q, want empty",
				seat, kamichaZoneWidth, got)
		}
	}
}

// TestRenderOpponentMeldsFitsOnOneLine asserts that a seat with one pon
// returns a single-line block whose width fits within the zone budget
// and contains the expected glyph + seat-source marker. Per the
// one-line scenario in the Play Screen Layout requirement.
func TestRenderOpponentMeldsFitsOnOneLine(t *testing.T) {
	g := game.New(7)
	g.SetTestMeld(game.SeatEast, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  game.SeatSouth,
	})
	m := NewWithGame(UnicodeRenderer{}, g)

	out := m.renderOpponentMelds(game.SeatEast, kamichaZoneWidth)
	if strings.Contains(out, "\n") {
		t.Errorf("expected single-line output for 1 pon; got multi-line:\n%s", out)
	}
	if w := lipgloss.Width(out); w > kamichaZoneWidth {
		t.Errorf("rendered width %d exceeds zone width %d. Output: %q",
			w, kamichaZoneWidth, out)
	}
	r := UnicodeRenderer{}
	p5Glyph := strings.Join(r.Tile(tile.Tile{ID: tile.P5}), "\n")
	if !strings.Contains(out, p5Glyph) {
		t.Errorf("output missing 5p glyph. Output: %q", out)
	}
	if !strings.Contains(out, "[S]") {
		t.Errorf("output missing [S] marker. Output: %q", out)
	}
}

// TestRenderOpponentMeldsWrapsWhenTooWide asserts that two melds whose
// combined width exceeds the zone budget wrap to two lines (one meld
// per line) within the zone budget. Per the wrap scenario.
func TestRenderOpponentMeldsWrapsWhenTooWide(t *testing.T) {
	g := game.New(7)
	g.SetTestMeld(game.SeatEast, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  game.SeatSouth,
	})
	g.SetTestMeld(game.SeatEast, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.M1}, {ID: tile.M1}, {ID: tile.M1}},
		From:  game.SeatWest,
	})
	m := NewWithGame(UnicodeRenderer{}, g)

	out := m.renderOpponentMelds(game.SeatEast, kamichaZoneWidth)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after wrap; got %d. Output:\n%s", len(lines), out)
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w > kamichaZoneWidth {
			t.Errorf("line %d width %d exceeds zone width %d. Line: %q",
				i, w, kamichaZoneWidth, line)
		}
	}
}

// TestRenderOpponentMeldsTruncatesWithKMore asserts that 4 ankans on a
// 20-col zone overflow even the 2-line wrap and produce a `+K more`
// suffix on a third line. Per the truncation scenario.
func TestRenderOpponentMeldsTruncatesWithKMore(t *testing.T) {
	g := game.New(7)
	for _, id := range []uint8{tile.EastWind, tile.SouthWind, tile.WestWind, tile.NorthWind} {
		g.SetTestMeld(game.SeatEast, game.Meld{
			Kind:    game.MeldKan,
			KanKind: game.KanAnkan,
			Tiles:   []tile.Tile{{ID: id}, {ID: id}, {ID: id}, {ID: id}},
		})
	}
	m := NewWithGame(UnicodeRenderer{}, g)

	out := m.renderOpponentMelds(game.SeatEast, kamichaZoneWidth)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (2 ankans + +K more); got %d. Output:\n%s",
			len(lines), out)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "+") || !strings.Contains(last, "more") {
		t.Errorf("third line missing `+K more` suffix. Line: %q", last)
	}
}

func TestRenderOpenMeldsForSeatRendersCorrectSeatMelds(t *testing.T) {
	g := game.New(7)
	// Plant a pon of M1 from South onto SeatEast.
	g.SetTestMeld(game.SeatEast, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.M1}, {ID: tile.M1}, {ID: tile.M1}},
		From:  game.SeatSouth,
	})
	// Plant an ankan of EastWind onto SeatNorth.
	g.SetTestMeld(game.SeatNorth, game.Meld{
		Kind:    game.MeldKan,
		KanKind: game.KanAnkan,
		Tiles: []tile.Tile{
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
			{ID: tile.EastWind},
		},
	})

	m := NewWithGame(UnicodeRenderer{}, g)

	r := UnicodeRenderer{}
	m1Glyph := strings.Join(r.Tile(tile.Tile{ID: tile.M1}), "\n")
	eastGlyph := strings.Join(r.Tile(tile.Tile{ID: tile.EastWind}), "\n")

	eastOut := m.renderOpenMeldsForSeat(game.SeatEast)
	if !strings.Contains(eastOut, m1Glyph) {
		t.Errorf("renderOpenMeldsForSeat(SeatEast) missing 1m glyph. Output:\n%s", eastOut)
	}
	if !strings.Contains(eastOut, "[S]") {
		t.Errorf(
			"renderOpenMeldsForSeat(SeatEast) missing [S] marker (called from South). Output:\n%s",
			eastOut,
		)
	}

	northOut := m.renderOpenMeldsForSeat(game.SeatNorth)
	if !strings.Contains(northOut, eastGlyph) {
		t.Errorf("renderOpenMeldsForSeat(SeatNorth) missing EastWind glyph. Output:\n%s", northOut)
	}
	if !strings.Contains(northOut, "[A]") {
		t.Errorf(
			"renderOpenMeldsForSeat(SeatNorth) missing [A] marker for ankan. Output:\n%s",
			northOut,
		)
	}

	// Cross-seat isolation: SeatEast's output must NOT contain SeatNorth's tiles.
	if strings.Contains(eastOut, eastGlyph) {
		t.Errorf(
			"renderOpenMeldsForSeat(SeatEast) leaked SeatNorth's EastWind tiles. Output:\n%s",
			eastOut,
		)
	}
}

func TestRenderHandWrapsMeldsWhenRowExceeds80Columns(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.S9},
		{ID: tile.M4},
		{ID: tile.M5},
		{ID: tile.SouthWind},
		{ID: tile.SouthWind},
	})
	for _, id := range []uint8{tile.EastWind, tile.SouthWind, tile.WestWind, tile.NorthWind} {
		g.SetTestMeld(game.SeatSouth, game.Meld{
			Kind:    game.MeldKan,
			KanKind: game.KanAnkan,
			Tiles:   []tile.Tile{{ID: id}, {ID: id}, {ID: id}, {ID: id}},
		})
	}

	m := NewWithGame(UnicodeRenderer{}, g)
	out := m.renderHand()

	if !strings.Contains(out, "\n") {
		t.Fatalf(
			"renderHand with 4 ankans + 13 concealed should wrap onto a second line. Output:\n%s",
			out,
		)
	}
	for line := range strings.SplitSeq(out, "\n") {
		if w := lipgloss.Width(line); w > targetWidth {
			t.Errorf(
				"rendered line width %d exceeds targetWidth %d. Line: %q",
				w,
				targetWidth,
				line,
			)
		}
	}
}
