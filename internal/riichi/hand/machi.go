package hand

import "github.com/benny123tw/mahjong-cli/internal/riichi/tile"

// Machi returns the sorted tile IDs that complete a 13-tile hand into any
// valid agari shape. Returns nil for hands of incorrect size.
func Machi(h Hand) []uint8 {
	if len(h.Concealed) != 13 {
		return nil
	}
	var winners []uint8
	counts := h.Counts()
	for id := range uint8(tile.TileCount) {
		if counts[id] >= 4 {
			continue
		}
		test := append([]tile.Tile(nil), h.Concealed...)
		test = append(test, tile.Tile{ID: id})
		if IsWinning(Hand{Concealed: test}) {
			winners = append(winners, id)
		}
	}
	return winners
}
