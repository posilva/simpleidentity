package main

import (
	"context"
	"fmt"
	"os"

	"github.com/posilva/simpleidentity/cmd"
)

func main() {
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
