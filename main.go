package main

import (
	"fmt"
	"os"

	"github.com/benny123tw/mahjong-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
