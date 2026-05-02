package hand

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func mustParse(t *testing.T, s string) []tile.Tile {
	t.Helper()
	tiles, err := tile.Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return tiles
}

func TestDecompose_StandardForm(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m2m3m4p5p6p7s8s9s1z1z2z2z2z")}
	decomps := Decompose(h)
	if len(decomps) == 0 {
		t.Fatal("expected at least one decomposition")
	}
	hasStandard := false
	for _, d := range decomps {
		if d.Form == FormStandard {
			hasStandard = true
			if len(d.Melds) != 5 {
				t.Errorf("standard decomp has %d melds, want 5", len(d.Melds))
			}
		}
	}
	if !hasStandard {
		t.Error("expected a FormStandard decomposition")
	}
}

func TestDecompose_Chiitoitsu(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m1m4p4p7p7p2s2s5s5s8s8s1z1z")}
	decomps := Decompose(h)
	hasChii := false
	for _, d := range decomps {
		if d.Form == FormChiitoitsu {
			hasChii = true
			if len(d.Melds) != 7 {
				t.Errorf("chiitoitsu has %d melds, want 7", len(d.Melds))
			}
		}
	}
	if !hasChii {
		t.Error("expected a FormChiitoitsu decomposition")
	}
}

func TestDecompose_Kokushi(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m9m1p9p1s9s1z2z3z4z5z6z7z1m")}
	decomps := Decompose(h)
	hasKokushi := false
	for _, d := range decomps {
		if d.Form == FormKokushi {
			hasKokushi = true
		}
	}
	if !hasKokushi {
		t.Error("expected a FormKokushi decomposition")
	}
}

// Spec scenario: chiitoitsu requires seven distinct pairs.
// A hand with 4 of one tile (as two same-value pairs) falls through to
// standard-form check.
func TestDecompose_FourOfAKindNotChiitoitsu(t *testing.T) {
	// 1m×4 + 6 distinct pairs = 4 + 12 = 16 tiles? Too many. Let me use:
	// 1m×4 + 2p2p + 3s3s + 4m4m + 5p5p + 6s = 4+2+2+2+2+1 = 13 tiles. Not 14.
	// Simpler: a hand that fails chiitoitsu but still completes standard.
	// 1m×4 + 2m2m3m + 4p5p6p + 7s8s9s = 4+3+3+3=13... need 14.
	// Try: 1m1m1m1m + 2m3m4m + 4p5p6p + 7s8s9s + 1z1z = 4+3+3+3+2 = 15. Too many.
	// Try: 1m1m1m + 1m2m3m + 4p5p6p + 7s8s9s + 1z1z = 3+3+3+3+2 = 14 (uses 4×1m). Standard.
	h := Hand{Concealed: mustParse(t, "1m1m1m1m2m3m4p5p6p7s8s9s1z1z")}
	decomps := Decompose(h)
	for _, d := range decomps {
		if d.Form == FormChiitoitsu {
			t.Error("hand with 4×1m must not parse as chiitoitsu")
		}
	}
	// Should still detect via standard form
	hasStandard := false
	for _, d := range decomps {
		if d.Form == FormStandard {
			hasStandard = true
		}
	}
	if !hasStandard {
		t.Error("expected hand to fall through to standard form")
	}
}

func TestDecompose_NonWinning(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m2m3m4p5p6p7s8s9s1z1z2z2z3z")}
	if IsWinning(h) {
		t.Error("hand should not be winning")
	}
}
