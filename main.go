package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"morsemanual/internal/application"
)

const databasePath = "logbook.db"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := application.Run(ctx, databasePath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
