package main

import (
	"fmt"
	"os"

	jev "jev-playground"
)

func main() {
	if err := jev.Main(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
