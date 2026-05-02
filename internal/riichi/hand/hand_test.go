package hand

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestHandZeroValueCalledMeldsIsEmpty(t *testing.T) {
	concealed := []tile.Tile{{ID: tile.M1}, {ID: tile.M2}, {ID: tile.M3}}
	h := Hand{Concealed: concealed}
	if got := len(h.Concealed); got != 3 {
		t.Errorf("Hand.Concealed len = %d, want 3", got)
	}
	if got := len(h.CalledMelds); got != 0 {
		t.Errorf("zero-value Hand.CalledMelds len = %d, want 0", got)
	}
}
