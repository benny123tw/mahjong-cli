// Package game implements the riichi mahjong game state machine: wall
// construction, dealing, turn cycle, call resolution, bot decisions, and the
// event log consumed by the TUI and golden-game tests.
//
// The package depends on internal/riichi/{tile,hand,yaku,calc} for rules
// evaluation but MUST NOT import internal/play, cmd/, bubbletea, or lipgloss.
// This keeps the game loop testable as plain `go test` and allows future
// changes (network play, headless harnesses) to embed the same engine.
package game

import (
	"math/rand/v2"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// numSeats is the four-player riichi convention.
const numSeats = 4

// deadWallSize is the standard 14-tile king's wall reserved for kan
// replacements and dora indicators. Live wall after dealing four 13-tile
// hands is 136 − 52 − 14 = 70 tiles.
const deadWallSize = 14

// Wall is a 136-tile shuffled stack with explicit live and dead regions.
// The PRNG is exposed via Rand() so the same seed drives both shuffle and bot
// probabilistic decisions, satisfying the deterministic-shuffle requirement.
type Wall struct {
	tiles     []tile.Tile
	drawIndex int
	rng       *rand.Rand
}

// NewWall builds a 136-tile wall (4 of each of 34 tile types, no red fives in
// v1) and shuffles it using a PCG PRNG seeded from `seed`. Two walls
// constructed with the same seed produce byte-identical tile orders.
func NewWall(seed int64) *Wall {
	tiles := make([]tile.Tile, 0, 34*4)
	for id := range uint8(tile.TileCount) {
		for range 4 {
			tiles = append(tiles, tile.Tile{ID: id})
		}
	}
	r := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9E3779B97F4A7C15))
	r.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
	return &Wall{tiles: tiles, rng: r}
}

// Rand returns the same PRNG used to shuffle the wall, so bot decisions
// reproduce deterministically alongside the shuffle.
func (w *Wall) Rand() *rand.Rand { return w.rng }

// allTiles returns the full 136-tile sequence in wall order. Test-only.
func (w *Wall) allTiles() []tile.Tile {
	out := make([]tile.Tile, len(w.tiles))
	copy(out, w.tiles)
	return out
}

// Deal removes 4×13 tiles from the wall front and returns one hand per seat
// (East, South, West, North). It also reveals the first dora indicator from
// the dead wall. The remaining 70 tiles form the live wall, drawable via
// Draw(); the dead wall (14 tiles, last in the slice) is reserved.
type DealResult struct {
	Hands         [numSeats][]tile.Tile
	DoraIndicator tile.Tile
}

func (w *Wall) Deal() DealResult {
	var hands [numSeats][]tile.Tile
	for seat := range numSeats {
		hands[seat] = make([]tile.Tile, 13)
		for i := range 13 {
			hands[seat][i] = w.tiles[w.drawIndex]
			w.drawIndex++
		}
	}
	// First dora indicator is the third tile from the top of the dead wall —
	// for v1 we just take the last tile of the wall as a stable proxy. Real
	// riichi positions it at a specific dead-wall slot; this simplification
	// stays correct for "single dora indicator revealed at deal time".
	indicator := w.tiles[len(w.tiles)-1]
	return DealResult{Hands: hands, DoraIndicator: indicator}
}

// Draw pops one tile from the front of the live wall. Returns ok=false once
// the live wall is exhausted (dead wall is reserved and never draws via Draw).
func (w *Wall) Draw() (tile.Tile, bool) {
	live := len(w.tiles) - deadWallSize
	if w.drawIndex >= live {
		return tile.Tile{}, false
	}
	t := w.tiles[w.drawIndex]
	w.drawIndex++
	return t, true
}

// LiveRemaining returns the number of tiles still drawable from the live wall.
func (w *Wall) LiveRemaining() int {
	live := len(w.tiles) - deadWallSize
	return live - w.drawIndex
}
