package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tom-redmine/go-redmine-cli/internal/redminecmd"
)

func main() {
	app := redminecmd.Run()
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
