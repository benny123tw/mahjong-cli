package play

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestKanKeyDeclaresAnkanWhenAvailable(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.NorthWind},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	mu := updated.(Model)

	melds := mu.game.Melds(game.SeatSouth)
	if len(melds) != 1 {
		t.Fatalf("after K key, melds count = %d, want 1", len(melds))
	}
	if melds[0].Kind != game.MeldKan || melds[0].KanKind != game.KanAnkan {
		t.Errorf(
			"after K, meld kind = (%d, %d), want (MeldKan, KanAnkan)",
			melds[0].Kind,
			melds[0].KanKind,
		)
	}
}

func TestKanKeyGreyedDuringRiichi(t *testing.T) {
	g := game.New(7)
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.S5},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.NorthWind},
	})
	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})
	g.SetTestRiichiDeclared(game.SeatSouth, true)

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	mu := updated.(Model)

	if len(mu.game.Melds(game.SeatSouth)) != 0 {
		t.Errorf("kan declared while in riichi; melds: %+v", mu.game.Melds(game.SeatSouth))
	}
	if mu.AckText() == "" {
		t.Errorf("ackText empty after rejected kan-during-riichi; want a reason string")
	}
}

func TestKanKeySubmitsMinkanInClaimWindow(t *testing.T) {
	g := game.New(7)
	// Human has 3 of 5p in hand, ready to minkan East's 5p discard.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.NorthWind},
	})
	g.SetTestState(game.StateAwaitingClaims{
		Discard:   tile.Tile{ID: tile.P5},
		Discarder: game.SeatEast,
	})

	m := NewWithGame(UnicodeRenderer{}, g)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	mu := updated.(Model)

	if st, ok := mu.game.State().(game.StateAwaitingDiscard); !ok || st.Player != game.SeatSouth {
		t.Errorf("after K minkan, state = %v, want AwaitingDiscard{South}", mu.game.State())
	}
	melds := mu.game.Melds(game.SeatSouth)
	if len(melds) != 1 || melds[0].Kind != game.MeldKan || melds[0].KanKind != game.KanMinkan {
		t.Errorf("after K minkan, melds = %+v, want one KanMinkan", melds)
	}
}

func TestBotRonsOnHumanShouminkanChankan(t *testing.T) {
	g := game.New(7)
	// Human (South) has open MeldPon for 5p and a 5p in hand for shouminkan.
	g.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.P5},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.EastWind},
		{ID: tile.SouthWind},
		{ID: tile.WestWind},
		{ID: tile.NorthWind},
	})
	// Plant the open MeldPon directly via the engine's internal field. Since
	// the play package doesn't have a setter, we go through the engine's
	// SetTestPond/SetTestHand pattern — but for melds there's no setter.
	// We use a small detour: drive a pon via the engine. To keep the test
	// simple, instead of pon, we acknowledge that this scenario requires
	// adding a SetTestMeld helper or reusing the engine's test path.
	// Use a SetTestMeld helper added below.
	g.SetTestMeld(game.SeatSouth, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  game.SeatEast,
	})

	// SeatNorth tenpai winning on 5p (kanchan 4p+6p).
	g.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M1},
		{ID: tile.M1},
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.P4},
		{ID: tile.P6},
	})

	g.SetTestState(game.StateAwaitingDiscard{Player: game.SeatSouth})

	m := NewWithGame(UnicodeRenderer{}, g)
	// Human presses K → shouminkan declared, state transitions to AwaitingChankan.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	mu := updated.(Model)
	if _, ok := mu.game.State().(game.StateAwaitingChankan); !ok {
		t.Fatalf("state after human K = %T, want AwaitingChankan", mu.game.State())
	}

	// Bot tick fires the chankan dispatcher, which submits the bot's ron.
	updated2, _ := mu.Update(BotTickMsg{})
	mu2 := updated2.(Model)
	st, ok := mu2.game.State().(game.StateRoundOver)
	if !ok {
		t.Fatalf("state after bot tick on chankan = %T, want StateRoundOver", mu2.game.State())
	}
	out, ok := st.Outcome.(game.OutcomeRon)
	if !ok {
		t.Fatalf("outcome after bot chankan ron = %T, want OutcomeRon", st.Outcome)
	}
	if out.Winner != game.SeatNorth {
		t.Errorf("chankan ron winner = %d, want SeatNorth", out.Winner)
	}
	if out.Loser != game.SeatSouth {
		t.Errorf("chankan ron loser = %d, want SeatSouth (declarer)", out.Loser)
	}
	hasChankan := false
	for _, ym := range out.Result.YakuMatches {
		if ym.Name == "Chankan" {
			hasChankan = true
			break
		}
	}
	if !hasChankan {
		yakuNames := make([]string, 0, len(out.Result.YakuMatches))
		for _, ym := range out.Result.YakuMatches {
			yakuNames = append(yakuNames, ym.Name)
		}
		t.Errorf("chankan yaku missing from bot ron result; got: %v", yakuNames)
	}
}
