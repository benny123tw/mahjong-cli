package game

import (
	"errors"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestAnkanSucceedsWithFourMatchingTiles(t *testing.T) {
	g := New(7)
	// Plant a 14-tile hand with four 5p, padded with arbitrary garbage.
	g.testSetHand(SeatEast, []tile.Tile{
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
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})

	doraBefore := len(g.DoraIndicators())
	if _, err := g.Step(InputDeclareAnkan{TileID: tile.P5}); err != nil {
		t.Fatalf("Step(ankan) returned err: %v", err)
	}

	// Hand: 14 - 4 + 1 (rinshan) = 11.
	if got := len(g.Hand(SeatEast)); got != 11 {
		t.Errorf("post-ankan hand size = %d, want 11", got)
	}
	melds := g.Melds(SeatEast)
	if len(melds) != 1 {
		t.Fatalf("melds count = %d, want 1", len(melds))
	}
	if melds[0].Kind != MeldKan || melds[0].KanKind != KanAnkan {
		t.Errorf("meld kind = (%d, %d), want (MeldKan, KanAnkan)", melds[0].Kind, melds[0].KanKind)
	}
	if len(g.DoraIndicators()) != doraBefore+1 {
		t.Errorf(
			"dora indicator count after ankan = %d, want %d",
			len(g.DoraIndicators()),
			doraBefore+1,
		)
	}
	if _, ok := g.State().(StateAwaitingDiscard); !ok {
		t.Errorf("state after ankan = %T, want StateAwaitingDiscard", g.State())
	}
	if !g.lastDrawWasRinshan[SeatEast] {
		t.Errorf("lastDrawWasRinshan[East] = false after ankan, want true")
	}
}

func TestAnkanRejectedWithThreeMatchingTiles(t *testing.T) {
	g := New(7)
	g.testSetHand(SeatEast, []tile.Tile{
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
		{ID: tile.Haku},
	})
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})

	_, err := g.Step(InputDeclareAnkan{TileID: tile.P5})
	if !errors.Is(err, ErrIllegalKan) {
		t.Errorf("Step(ankan with 3 tiles) err = %v, want ErrIllegalKan", err)
	}
	if len(g.Melds(SeatEast)) != 0 {
		t.Errorf("melds appended on rejected ankan: %v", g.Melds(SeatEast))
	}
}

func TestKanRejectedWhenInRiichi(t *testing.T) {
	g := New(7)
	g.testSetHand(SeatEast, []tile.Tile{
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
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})
	g.SetTestRiichiDeclared(SeatEast, true)

	_, err := g.Step(InputDeclareAnkan{TileID: tile.P5})
	if !errors.Is(err, ErrIllegalKan) {
		t.Errorf("ankan during riichi err = %v, want ErrIllegalKan", err)
	}
}

func TestMinkanWinsOverPon(t *testing.T) {
	g := New(7)
	// SeatSouth has 3 of 5p (kan-eligible), SeatNorth has 2 of 5p (pon-eligible).
	g.testSetHand(SeatSouth, []tile.Tile{
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
	g.testSetHand(SeatNorth, []tile.Tile{
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
		{ID: tile.Haku},
	})
	g.testSetState(StateAwaitingClaims{Discard: tile.Tile{ID: tile.P5}, Discarder: SeatEast})

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		SeatSouth: {Kind: ClaimKan},
		SeatNorth: {Kind: ClaimPon},
	}})
	if err != nil {
		t.Fatalf("Step(kan+pon claims) err: %v", err)
	}
	melds := g.Melds(SeatSouth)
	if len(melds) != 1 || melds[0].Kind != MeldKan || melds[0].KanKind != KanMinkan {
		t.Errorf("South melds = %+v, want one KanMinkan", melds)
	}
	if len(g.Melds(SeatNorth)) != 0 {
		t.Errorf("North melds = %+v, want empty (pon lost to kan)", g.Melds(SeatNorth))
	}
	if st, ok := g.State().(StateAwaitingDiscard); !ok || st.Player != SeatSouth {
		t.Errorf("state after minkan = %v, want AwaitingDiscard{South}", g.State())
	}
}

func TestShouminkanCompletesWhenNoChankan(t *testing.T) {
	g := New(7)
	// SeatSouth has open MeldPon for 5p, plus a 5p in hand for upgrade.
	g.testSetHand(SeatSouth, []tile.Tile{
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
	g.melds[SeatSouth] = []Meld{{
		Kind:  MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  SeatEast,
	}}
	g.testSetState(StateAwaitingDiscard{Player: SeatSouth})

	if _, err := g.Step(InputDeclareShouminkan{TileID: tile.P5}); err != nil {
		t.Fatalf("Step(shouminkan) returned err: %v", err)
	}
	if _, ok := g.State().(StateAwaitingChankan); !ok {
		t.Fatalf("state after shouminkan declaration = %T, want AwaitingChankan", g.State())
	}

	// No ron — pass.
	if _, err := g.Step(InputResolveClaims{Claims: nil}); err != nil {
		t.Fatalf("Step(pass chankan) returned err: %v", err)
	}
	melds := g.Melds(SeatSouth)
	if len(melds) != 1 || melds[0].KanKind != KanShouminkan {
		t.Errorf("South meld after shouminkan = %+v, want KanShouminkan", melds)
	}
	if len(melds[0].Tiles) != 4 {
		t.Errorf("upgraded meld tile count = %d, want 4", len(melds[0].Tiles))
	}
	if st, ok := g.State().(StateAwaitingDiscard); !ok || st.Player != SeatSouth {
		t.Errorf("state after shouminkan completion = %v, want AwaitingDiscard{South}", g.State())
	}
}

func TestChankanRonPreemptsShouminkan(t *testing.T) {
	g := New(7)
	// SeatSouth has open MeldPon for 5p, plus a 5p in hand for the upgrade.
	g.testSetHand(SeatSouth, []tile.Tile{
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
	g.melds[SeatSouth] = []Meld{{
		Kind:  MeldPon,
		Tiles: []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
		From:  SeatEast,
	}}
	// SeatNorth is tenpai on 5p (kanchan 4p+6p), shape wins with 5p.
	// Hand: 1m1m + 234m + 234p + 234s + 4p+6p = 2+3+3+3+2 = 13.
	g.testSetHand(SeatNorth, []tile.Tile{
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

	g.testSetState(StateAwaitingDiscard{Player: SeatSouth})
	if _, err := g.Step(InputDeclareShouminkan{TileID: tile.P5}); err != nil {
		t.Fatalf("Step(shouminkan) returned err: %v", err)
	}
	if _, ok := g.State().(StateAwaitingChankan); !ok {
		t.Fatalf("expected AwaitingChankan, got %T", g.State())
	}

	_, err := g.Step(InputResolveClaims{Claims: map[Seat]Claim{
		SeatNorth: {Kind: ClaimRon},
	}})
	if err != nil {
		t.Fatalf("Step(chankan ron) returned err: %v", err)
	}

	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after chankan ron = %T, want StateRoundOver", g.State())
	}
	out, ok := st.Outcome.(OutcomeRon)
	if !ok {
		t.Fatalf("outcome after chankan ron = %T, want OutcomeRon", st.Outcome)
	}
	if out.Winner != SeatNorth {
		t.Errorf("chankan ron winner = %d, want SeatNorth", out.Winner)
	}
	if out.Loser != SeatSouth {
		t.Errorf("chankan ron loser = %d, want SeatSouth (declarer)", out.Loser)
	}
	if out.Tile.ID != tile.P5 {
		t.Errorf("chankan ron tile = %s, want 5p", out.Tile)
	}
	// Verify chankan yaku appears in the result.
	hasChankan := false
	for _, m := range out.Result.YakuMatches {
		if m.Name == "Chankan" {
			hasChankan = true
			break
		}
	}
	if !hasChankan {
		yakuNames := make([]string, 0, len(out.Result.YakuMatches))
		for _, m := range out.Result.YakuMatches {
			yakuNames = append(yakuNames, m.Name)
		}
		t.Errorf("chankan yaku not in result; got yaku list: %v", yakuNames)
	}
	// Verify the pon meld was NOT upgraded (chankan pre-empts the kan).
	melds := g.Melds(SeatSouth)
	if len(melds) != 1 || melds[0].Kind != MeldPon || melds[0].KanKind != KanNone {
		t.Errorf("South's pon was modified after pre-empted shouminkan: %+v", melds)
	}
}

func TestRinshanTsumoSetsRinshanFlag(t *testing.T) {
	g := New(7)
	// Post-ankan-and-rinshan state planted directly:
	//   - Concealed hand: 234m + 234p + 234s + 2m + 2m (11 tiles).
	//   - Open ankan meld of 5p (4 tiles, contributes 3 to the 14-tile shape).
	//   - lastDrawWasRinshan flag set so contextForWin populates Rinshan.
	// Expanded view: 234m+234p+234s+2m2m + 5p5p5p (kan as triplet) = 14 tiles.
	// Winning shape: 4 sets (3 runs + 1 ankan-triplet) + 2m2m pair. Winning
	// tile is the rightmost (the rinshan replacement, planted as 2m).
	g.testSetHand(SeatEast, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M2}, // pair-completing rinshan replacement at index 10
		{ID: tile.M2}, // 2nd 2m, the original pair
	})
	g.melds[SeatEast] = []Meld{{
		Kind:    MeldKan,
		KanKind: KanAnkan,
		Tiles:   []tile.Tile{{ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}, {ID: tile.P5}},
	}}
	g.lastDrawWasRinshan[SeatEast] = true
	g.testSetState(StateAwaitingDiscard{Player: SeatEast})

	_, err := g.Step(InputDeclareTsumo{})
	if err != nil {
		t.Fatalf("Step(tsumo on rinshan) returned err: %v", err)
	}

	st, ok := g.State().(StateRoundOver)
	if !ok {
		t.Fatalf("state after rinshan tsumo = %T, want StateRoundOver", g.State())
	}
	out, ok := st.Outcome.(OutcomeTsumo)
	if !ok {
		t.Fatalf("outcome after rinshan tsumo = %T, want OutcomeTsumo", st.Outcome)
	}
	hasRinshan := false
	for _, m := range out.Result.YakuMatches {
		if m.Name == "Rinshan kaihou" {
			hasRinshan = true
			break
		}
	}
	if !hasRinshan {
		yakuNames := make([]string, 0, len(out.Result.YakuMatches))
		for _, m := range out.Result.YakuMatches {
			yakuNames = append(yakuNames, m.Name)
		}
		t.Errorf("rinshan kaihou yaku not in result; got yaku list: %v", yakuNames)
	}
}
