package main

import (
	"fmt"
	"os"

	"github.com/toumorokoshi/aep-sandbox/aepc/cmd"aepc
)

func main() {
	if err := cmd.NewCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
