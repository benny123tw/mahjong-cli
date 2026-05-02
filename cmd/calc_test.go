package cmd

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func runWithArgs(t *testing.T, args []string) (stdout, stderr string, err error) {
	t.Helper()
	flagSeat = "S"
	flagRound = "E"
	flagRiichi = false
	flagTsumo = false
	flagDora = nil
	flagUradora = nil

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("..", "testdata", "calc", "golden", name+".txt")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run `go test -update` to create): %v", path, err)
	}
	if got != string(want) {
		t.Errorf(
			"output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s",
			name,
			got,
			string(want),
		)
	}
}

func TestCalc_Golden_WinningHandSeatW(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{
		"calc", "1m2m3m4p5p6p7s8s9s1z1z2z2z2z",
		"--tsumo", "--seat", "W",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	compareGolden(t, "winning_seat_w_tsumo", stdout)
}

func TestCalc_Golden_TenpaiTanki(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{
		"calc", "1m2m3m4m5m6m7p8p9p1s2s3s1z",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	compareGolden(t, "tenpai_tanki", stdout)
}

func TestCalc_Golden_TenpaiNotShanten2(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{
		"calc", "1m2m3m4p5p6p7s8s9s4z5z6z7z",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	compareGolden(t, "shanten2_not_tenpai", stdout)
}

// Smoke-test hand from the discussion that triggered Group A. The hand
// 1m1m1m4m4m4m7m7m7m9m9m9m5m5m is four concealed triplets plus a 5m pair,
// tanki-ron on 5m — by full riichi rules that's Suuankou yakuman (the
// kan-aware Group D detector now fires; pre-Group-D this reported
// chinitsu + toitoi + sanankou as constituent normal-han yaku).
func TestCalc_Golden_SmokeTestChinitsuToitoiSanankou(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{
		"calc", "1m1m1m4m4m4m7m7m7m9m9m9m5m5m",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	compareGolden(t, "smoke_chinitsu_toitoi_sanankou", stdout)
}

func TestCalc_InvalidHand(t *testing.T) {
	_, _, err := runWithArgs(t, []string{
		"calc", "0z" + strings.Repeat("1m", 13),
	})
	if err == nil {
		t.Fatal("expected error for invalid hand")
	}
	if !strings.Contains(err.Error(), "parse hand") {
		t.Errorf("error = %v, want it to mention 'parse hand'", err)
	}
}
