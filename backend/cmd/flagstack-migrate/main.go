package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flagstack/flagstack/backend/internal/config"
	"github.com/flagstack/flagstack/backend/internal/database"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fail(fmt.Errorf("open database: %w", err))
	}
	defer pool.Close()

	client := database.NewEntClient(pool)
	defer client.Close()

	if err := database.Migrate(ctx, pool, client); err != nil {
		fail(err)
	}
	fmt.Println("FlagStack database schema is up to date.")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
