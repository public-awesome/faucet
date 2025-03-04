package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/public-awesome/faucet/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server, err := server.NewServer(log)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	server.Run(ctx)

}
