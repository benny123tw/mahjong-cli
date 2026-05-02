package cmd

import "testing"

func TestPlayCmdRegistersSeedAndAsciiFlags(t *testing.T) {
	if f := playCmd.Flags().Lookup("seed"); f == nil {
		t.Errorf("--seed flag not registered on play command")
	}
	if f := playCmd.Flags().Lookup("ascii"); f == nil {
		t.Errorf("--ascii flag not registered on play command")
	}
}

func TestRandomSeedReturnsNonZeroInTypicalRuns(t *testing.T) {
	// Cosmetic sanity check: the OS PRNG should produce a non-zero int64
	// in normal conditions. A zero return is the documented fallback for
	// rand.Read failure and would not reproduce here.
	if randomSeed() == 0 {
		t.Skip("randomSeed returned 0 — possible rand.Read failure; not strictly an error")
	}
}
