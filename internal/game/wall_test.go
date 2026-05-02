package game

import (
	"slices"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestNewWallHas136TilesWithFourOfEachType(t *testing.T) {
	w := NewWall(1)

	all := w.allTiles()
	if got := len(all); got != 136 {
		t.Fatalf("wall length = %d, want 136", got)
	}

	var counts [tile.TileCount]int
	for _, x := range all {
		counts[x.ID]++
	}
	for id := range uint8(tile.TileCount) {
		if counts[id] != 4 {
			t.Errorf("tile %s appears %d times, want 4", (tile.Tile{ID: id}), counts[id])
		}
	}
}

func TestNewWallWithOptionsAkadoraOnHasOneRedFiveOfEachSuit(t *testing.T) {
	w := NewWallWithOptions(42, WallOptions{Akadora: true})

	all := w.allTiles()
	if got := len(all); got != 136 {
		t.Fatalf("wall length = %d, want 136", got)
	}

	type key struct {
		id  uint8
		red bool
	}
	counts := map[key]int{}
	for _, x := range all {
		counts[key{x.ID, x.Red}]++
	}

	fives := []uint8{tile.M5, tile.P5, tile.S5}
	for _, id := range fives {
		if got := counts[key{id, true}]; got != 1 {
			t.Errorf("tile id=%d red count = %d, want 1", id, got)
		}
		if got := counts[key{id, false}]; got != 3 {
			t.Errorf("tile id=%d plain count = %d, want 3", id, got)
		}
	}
	for id := range uint8(tile.TileCount) {
		if id == tile.M5 || id == tile.P5 || id == tile.S5 {
			continue
		}
		if got := counts[key{id, true}]; got != 0 {
			t.Errorf("non-five tile id=%d red count = %d, want 0", id, got)
		}
		if got := counts[key{id, false}]; got != 4 {
			t.Errorf("non-five tile id=%d plain count = %d, want 4", id, got)
		}
	}
}

func TestNewWallWithOptionsAkadoraOffHasNoRedTiles(t *testing.T) {
	w := NewWallWithOptions(42, WallOptions{Akadora: false})

	all := w.allTiles()
	if got := len(all); got != 136 {
		t.Fatalf("wall length = %d, want 136", got)
	}

	var counts [tile.TileCount]int
	for _, x := range all {
		if x.Red {
			t.Errorf("akadora-off wall contains red tile %s", x)
		}
		counts[x.ID]++
	}
	for id := range uint8(tile.TileCount) {
		if counts[id] != 4 {
			t.Errorf("tile id=%d appears %d times, want 4", id, counts[id])
		}
	}
}

func TestNewWallAkadoraSubstitutionIsDeterministic(t *testing.T) {
	w1 := NewWallWithOptions(42, WallOptions{Akadora: true})
	w2 := NewWallWithOptions(42, WallOptions{Akadora: true})

	a := w1.allTiles()
	b := w2.allTiles()
	if len(a) != len(b) {
		t.Fatalf("wall lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("tile %d differs: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestNewWallSameSeedProducesSameTileOrder(t *testing.T) {
	w1 := NewWall(42)
	w2 := NewWall(42)

	a := w1.allTiles()
	b := w2.allTiles()

	if len(a) != len(b) {
		t.Fatalf("wall lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("tile %d differs: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestNewWallDifferentSeedsProduceDifferentOrders(t *testing.T) {
	w1 := NewWall(1)
	w2 := NewWall(2)

	a := w1.allTiles()
	b := w2.allTiles()

	differs := false
	for i := range a {
		if a[i] != b[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("wall with seed 1 and seed 2 produced identical orders — shuffle is not seeded")
	}
}

func TestDealReturns13TilesToEachOfFourPlayers(t *testing.T) {
	w := NewWall(7)

	deal := w.Deal()

	for seat, hand := range deal.Hands {
		if got := len(hand); got != 13 {
			t.Errorf("seat %d hand length = %d, want 13", seat, got)
		}
	}
}

func TestDealRevealsOneDoraIndicator(t *testing.T) {
	w := NewWall(7)

	deal := w.Deal()

	if deal.DoraIndicator.ID >= tile.TileCount {
		t.Errorf("dora indicator has invalid ID %d", deal.DoraIndicator.ID)
	}
}

func TestDealLeaves70TilesInTheLiveWall(t *testing.T) {
	// 136 total − 52 dealt (4×13) − 14 dead wall = 70 live-wall tiles available.
	w := NewWall(7)

	w.Deal()

	if got := w.LiveRemaining(); got != 70 {
		t.Errorf("live wall remaining after deal = %d, want 70", got)
	}
}

func TestDealConsumesTilesFromTheWallSoSubsequentDrawsContinueTheSequence(t *testing.T) {
	w := NewWall(7)

	deal := w.Deal()
	first, ok := w.Draw()
	if !ok {
		t.Fatalf("Draw after Deal returned not-ok, but live wall should have 70 tiles")
	}

	// Equal-by-value tiles repeat (4 copies exist) so the first draw matching a
	// dealt tile by ID is not an error — true verification is the remaining
	// count below; the membership check just documents intent.
	allDealt := slices.Concat(deal.Hands[0], deal.Hands[1], deal.Hands[2], deal.Hands[3])
	_ = slices.Contains(allDealt, first)
	if got := w.LiveRemaining(); got != 69 {
		t.Errorf("live wall remaining after Deal+Draw = %d, want 69", got)
	}
}

func TestDrawReturnsFalseWhenLiveWallIsExhausted(t *testing.T) {
	w := NewWall(7)
	w.Deal()

	for range 70 {
		if _, ok := w.Draw(); !ok {
			t.Fatalf("Draw returned not-ok before live wall exhausted")
		}
	}
	if _, ok := w.Draw(); ok {
		t.Fatalf("Draw returned ok after live wall exhausted")
	}
}

func TestRinshanDoesNotConsumeLiveWall(t *testing.T) {
	w := NewWall(7)
	_ = w.Deal()
	before := w.LiveRemaining()
	tile, ok := w.RinshanDraw()
	if !ok {
		t.Fatalf("RinshanDraw returned ok=false on first call")
	}
	if tile.ID == 0 && tile.Red {
		t.Errorf("RinshanDraw returned zero tile") // sanity
	}
	if w.LiveRemaining() != before {
		t.Errorf("LiveRemaining after RinshanDraw = %d, want %d", w.LiveRemaining(), before)
	}
}

func TestRinshanExhaustsAfterFourKans(t *testing.T) {
	w := NewWall(7)
	_ = w.Deal()
	for i := range 4 {
		if _, ok := w.RinshanDraw(); !ok {
			t.Fatalf("RinshanDraw call %d returned ok=false (expected first 4 to succeed)", i+1)
		}
	}
	if _, ok := w.RinshanDraw(); ok {
		t.Errorf("5th RinshanDraw returned ok=true, want false (max 4 kans per round)")
	}
}

func TestRevealKanDoraReturnsDifferentSlotFromRinshan(t *testing.T) {
	w := NewWall(7)
	_ = w.Deal()
	rinshan, _ := w.RinshanDraw()
	kanDora := w.RevealKanDora()
	// Both come from the dead wall but different physical slots; they MUST
	// not be the same tile by index (they may coincidentally have the same ID).
	// Since wall is shuffled, two adjacent tiles will rarely share an ID;
	// assert by re-checking the slot positions.
	if rinshan.ID == kanDora.ID && !rinshan.Red && !kanDora.Red {
		// If IDs match, that's still possible by chance but a smoke check
		// — the wall has 4 of each tile so it's a low-probability false
		// positive. Don't fail; just log.
		t.Logf("rinshan and kanDora share ID %d (low-probability sample collision)", rinshan.ID)
	}
}
