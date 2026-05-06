package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "mahjong",
	Short: "Riichi mahjong tools",
}

func Execute(info BuildInfo) error {
	buildInfo = info
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(calcCmd)
	rootCmd.AddCommand(versionCmd)
}
