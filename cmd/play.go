package cmd

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/play"
)

var (
	flagPlayASCII bool
	flagPlaySeed  int64
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Launch the riichi play screen (TUI)",
	Long: `Launch an interactive bubbletea TUI for a riichi hand against three
dummy bots. Use --seed to reproduce a specific game; without --seed the
program prints the OS-derived seed at startup so the run can be replayed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var renderer play.Renderer = play.UnicodeRenderer{}
		if flagPlayASCII {
			renderer = play.ASCIIRenderer{}
		}

		seed := flagPlaySeed
		if seed == 0 {
			seed = randomSeed()
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Seed: %d\n", seed)

		g := game.New(seed)
		m := play.NewWithGame(renderer, g)
		_, err := tea.NewProgram(m).Run()
		return err
	},
}

// randomSeed pulls 8 bytes from crypto/rand and folds them into an int64.
// Used when --seed is not provided so the game is reproducible after the
// fact via the printed seed.
func randomSeed() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Last-resort fallback: low-entropy seed based on a short loop. The
		// program still works deterministically; the user can pass --seed
		// instead if they hit this path.
		return 0xdeadbeef
	}
	return int64(
		binary.LittleEndian.Uint64(b[:]),
	) //nolint:gosec // seed is informational, not cryptographic
}

func init() {
	playCmd.Flags().
		BoolVar(&flagPlayASCII, "ascii", false, "Use ASCII boxed tile rendering instead of Unicode glyphs")
	playCmd.Flags().
		Int64Var(&flagPlaySeed, "seed", 0, "Deterministic shuffle seed (0 = OS random; printed at startup)")
	rootCmd.AddCommand(playCmd)
}
