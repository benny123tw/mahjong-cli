package play

import (
	"fmt"
	"strings"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

const (
	pondTilesPerRow     = 6
	pondMaxVisibleTiles = 12
)

// pondTileRenderer is the subset of Renderer used for pond tiles. Unicode
// renderer's 1-row glyph form is used directly; ASCII boxed renderer is
// substituted with ASCIIPondRenderer for the compact 1-row form.
type pondTileRenderer interface {
	Tile(t tile.Tile) []string
}

// pondRendererFor returns the pond-specific renderer compatible with `r`:
// Unicode passes through (already 1-row); ASCII boxed routes to compact.
func pondRendererFor(r Renderer) pondTileRenderer {
	if _, ok := r.(ASCIIRenderer); ok {
		return ASCIIPondRenderer{}
	}
	return r
}

// renderPondZone formats a per-seat discard zone: up to 12 most-recent
// discards in 6-wide sub-rows. Older discards scroll off the top with a
// `+N earlier` indicator. Returns empty when discards is empty.
func renderPondZone(discards []tile.Tile, r Renderer) string {
	pr := pondRendererFor(r)

	var lines []string

	overflow := 0
	visible := discards
	if len(visible) > pondMaxVisibleTiles {
		overflow = len(visible) - pondMaxVisibleTiles
		visible = visible[len(visible)-pondMaxVisibleTiles:]
	}

	if overflow > 0 {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("+%d earlier", overflow)))
	}

	for start := 0; start < len(visible); start += pondTilesPerRow {
		end := min(start+pondTilesPerRow, len(visible))
		row := visible[start:end]
		cells := make([]string, len(row))
		for i, t := range row {
			cells[i] = pr.Tile(t)[0]
		}
		lines = append(lines, strings.Join(cells, ""))
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
