package play

import (
	"strings"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestUnicodeRendererProducesGlyphWithVS15(t *testing.T) {
	r := UnicodeRenderer{}
	lines := r.Tile(tile.Tile{ID: tile.M1})
	if len(lines) != 1 {
		t.Fatalf("Unicode tile lines = %d, want 1", len(lines))
	}
	if !strings.Contains(lines[0], vs15) {
		t.Errorf("Unicode tile %q missing VS-15 (U+FE0E)", lines[0])
	}
}

func TestASCIIRendererProducesFourByThree(t *testing.T) {
	r := ASCIIRenderer{}
	lines := r.Tile(tile.Tile{ID: tile.M1})
	if len(lines) != 3 {
		t.Fatalf("ASCII tile lines = %d, want 3", len(lines))
	}
	for i, line := range lines {
		if got := visibleWidth(line); got != 4 {
			t.Errorf("ASCII tile line %d width = %d, want 4 (line=%q)", i, got, line)
		}
	}
}

func TestASCIIPondRendererProducesFourByOne(t *testing.T) {
	r := ASCIIPondRenderer{}
	lines := r.Tile(tile.Tile{ID: tile.M1})
	if len(lines) != 1 {
		t.Fatalf("ASCII compact tile lines = %d, want 1", len(lines))
	}
	if got := visibleWidth(lines[0]); got != 4 {
		t.Errorf("ASCII compact tile width = %d, want 4 (line=%q)", got, lines[0])
	}
}

func TestASCIIPondRendererFormatsTilesAsBracketedCanonical(t *testing.T) {
	r := ASCIIPondRenderer{}
	tests := []struct {
		t    tile.Tile
		want string
	}{
		{tile.Tile{ID: tile.M1}, "[1m]"},
		{tile.Tile{ID: tile.P5}, "[5p]"},
		{tile.Tile{ID: tile.S9}, "[9s]"},
		{tile.Tile{ID: tile.EastWind}, "[1z]"},
		{tile.Tile{ID: tile.Chun}, "[7z]"},
	}
	for _, tt := range tests {
		got := r.Tile(tt.t)[0]
		if got != tt.want {
			t.Errorf("ASCII compact tile for %s = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestASCIIPondRendererBackIsBracketedHash(t *testing.T) {
	r := ASCIIPondRenderer{}
	lines := r.Back()
	if len(lines) != 1 {
		t.Fatalf("ASCII compact back lines = %d, want 1", len(lines))
	}
	if got := visibleWidth(lines[0]); got != 4 {
		t.Errorf("ASCII compact back width = %d, want 4 (line=%q)", got, lines[0])
	}
}

// visibleWidth counts runes in `s` (proxy for visible width when there are no
// double-width or zero-width characters; sufficient for ASCII forms).
func visibleWidth(s string) int {
	return len([]rune(s))
}
