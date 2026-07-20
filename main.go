package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/moveeeax/envsync/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		if !errors.Is(err, cmd.ErrValidationFailed()) {
			fmt.Fprintln(os.Stderr, "envsync:", err)
		}
		os.Exit(1)
	}
}
