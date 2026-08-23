package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"ditdah/internal/application"

	"github.com/adrg/xdg"
)

const databaseFile = "ditdah/logbook.db"

var version = "development"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	databasePath, err := xdg.DataFile(databaseFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := application.Run(ctx, databasePath, version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
