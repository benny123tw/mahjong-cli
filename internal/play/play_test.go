package play

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
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
