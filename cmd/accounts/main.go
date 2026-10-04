package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mike-moseley/redoubt/internal/accounts"
	"github.com/mike-moseley/redoubt/internal/accounts/queries"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Error creating pgxpool: %v", err)
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Error talking to pgxpool: %v", err)
	}

	srv := accounts.NewServer(queries.New(pool))

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv.Routes(),
	}

	log.Printf("listening on %s", httpServer.Addr)
	log.Fatal(httpServer.ListenAndServe())
}
