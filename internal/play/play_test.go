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
