package hand

import (
	"slices"
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func tilesToStrings(ids []uint8) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = (tile.Tile{ID: id}).String()
	}
	return out
}

// Spec example table: shanten and machi by shape.
func TestShantenAndMachi_ExampleTable(t *testing.T) {
	cases := []struct {
		name        string
		hand        string
		wantShanten int
		wantMachi   []string
	}{
		{
			name:        "tanki on 1z (4 chii + lone honor)",
			hand:        "1m2m3m4m5m6m7p8p9p1s2s3s1z",
			wantShanten: 0,
			wantMachi:   []string{"1z"},
		},
		{
			name:        "shanpon on 1z and 2z",
			hand:        "1m2m3m4p5p6p7s8s9s1z1z2z2z",
			wantShanten: 0,
			wantMachi:   []string{"1z", "2z"},
		},
		{
			name:        "ryanmen on 1m and 4m",
			hand:        "2m3m4p5p6p7p8p9p1s2s3s1z1z",
			wantShanten: 0,
			wantMachi:   []string{"1m", "4m"},
		},
		{
			name:        "tanki on 1z (sou-pair shape)",
			hand:        "1m2m3m4p5p6p7s7s8s8s9s9s1z",
			wantShanten: 0,
			wantMachi:   []string{"1z"},
		},
		{
			name:        "non-tenpai (2-shanten)",
			hand:        "1m2m3m4p5p6p7s8s9s4z5z6z7z",
			wantShanten: 2,
			wantMachi:   nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := Hand{Concealed: mustParse(t, c.hand)}
			gotS := Shanten(h)
			if gotS != c.wantShanten {
				t.Errorf("Shanten(%s) = %d, want %d", c.hand, gotS, c.wantShanten)
			}
			gotM := Machi(h)
			gotMStr := tilesToStrings(gotM)
			slices.Sort(gotMStr)
			wantM := append([]string(nil), c.wantMachi...)
			slices.Sort(wantM)
			if !slices.Equal(gotMStr, wantM) {
				t.Errorf("Machi(%s) = %v, want %v", c.hand, gotMStr, wantM)
			}
		})
	}
}

func TestShanten_WinningHandIsNegativeOne(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m2m3m4p5p6p7s8s9s1z1z2z2z2z")}
	if got := Shanten(h); got != -1 {
		t.Errorf("Shanten = %d, want -1 for winning hand", got)
	}
}

func TestShanten_TenpaiHasShantenZero(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m2m3m4m5m6m7p8p9p1s2s3s1z")}
	if got := Shanten(h); got != 0 {
		t.Errorf("Shanten = %d, want 0", got)
	}
}

func TestShanten_NonTenpaiAtLeastOne(t *testing.T) {
	h := Hand{Concealed: mustParse(t, "1m2m3m4p5p6p7s8s9s4z5z6z7z")}
	if got := Shanten(h); got < 1 {
		t.Errorf("Shanten = %d, want >= 1", got)
	}
	if got := Machi(h); len(got) != 0 {
		t.Errorf("Machi = %v, want empty", got)
	}
}
