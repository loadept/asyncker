package main

import (
	"os"

	"loadept.com/pkg/asyncker/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
