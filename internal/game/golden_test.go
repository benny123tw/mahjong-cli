package game

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden controls whether golden test files are regenerated. Set via
// `go test -update ./internal/game/...`. The flag is defined here only when
// the testing binary parses it; regular `go test` runs without the flag and
// keeps strict diff behaviour.
var updateGolden = flag.Bool(
	"update",
	false,
	"regenerate golden game logs under testdata/game/golden",
)

// goldenLog is the JSON shape of a captured deterministic game.
type goldenLog struct {
	Seed   int64    `json:"seed"`
	Events []string `json:"events"`
}

// TestGoldenSeed runs a deterministic full round at the given seed,
// auto-discarding tile 0 each turn until the live wall exhausts (no calls,
// no claims). Captures the resulting transition log into
// testdata/game/golden/seed-<N>.json. With -update, the golden is rewritten;
// without -update, mismatches fail the test.
func TestGoldenSeed(t *testing.T) {
	const seed = 42
	g := New(seed)
	for range 70 {
		mustStepGolden(t, g, InputDraw{})
		mustStepGolden(t, g, InputDiscard{Index: 0})
		mustStepGolden(t, g, InputResolveClaims{Claims: nil})
	}
	mustStepGolden(t, g, InputDraw{}) // exhausts wall — ryuukyoku

	got := goldenLog{Seed: seed, Events: g.EventLog()}
	gotBytes, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden: %v", err)
	}

	path := filepath.Join("..", "..", "testdata", "game", "golden", "seed-42.json")
	if *updateGolden {
		if err := os.WriteFile(path, append(gotBytes, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden file %s", path)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		// First run: write the file and pass — author re-runs the suite to
		// commit the bytes.
		if err := os.WriteFile(path, append(gotBytes, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("created initial golden %s — re-run suite to commit", path)
		return
	}
	wantStr := string(want)
	gotStr := string(append(gotBytes, '\n'))
	if wantStr != gotStr {
		t.Errorf(
			"golden mismatch for %s — diff with the file or rerun with -update.\n--- got ---\n%s\n--- want ---\n%s",
			path,
			gotStr,
			wantStr,
		)
	}
}

func mustStepGolden(t *testing.T, g *Game, in Input) {
	t.Helper()
	if _, err := g.Step(in); err != nil {
		t.Fatalf("Step(%T): %v", in, err)
	}
}
