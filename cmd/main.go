package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vinimax001/finance-tracker/internal/finance"
	httpapi "github.com/vinimax001/finance-tracker/internal/http"

	_ "github.com/jackc/pgx/v5/stdlib" // driver pgx para database/sql
)

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	storage := getenv("STORAGE", "memory") // "postgres" | "memory"

	var repo finance.Repository
	if storage == "postgres" {
		// Tentar buscar credenciais do Secrets Manager primeiro
		secretName := os.Getenv("RDS_SECRET_NAME")
		var dsn string

		if secretName != "" {
			// Usar Secrets Manager para username e password
			log.Printf("Fetching RDS credentials from Secrets Manager: %s", secretName)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			secret, err := finance.GetRDSCredentials(ctx, secretName)
			if err != nil {
				log.Fatalf("failed to get RDS credentials from Secrets Manager: %v", err)
			}

			// Buscar host, port e dbname de variáveis de ambiente
			dbHost := os.Getenv("DB_HOST")
			if dbHost == "" {
				log.Fatalf("DB_HOST environment variable is required when using RDS_SECRET_NAME")
			}

			dbPortStr := getenv("DB_PORT", "5432")
			dbPort := 5432
			if _, err := fmt.Sscanf(dbPortStr, "%d", &dbPort); err != nil {
				log.Fatalf("invalid DB_PORT value: %v", err)
			}

			dbName := os.Getenv("DB_NAME")
			if dbName == "" {
				log.Fatalf("DB_NAME environment variable is required when using RDS_SECRET_NAME")
			}

			dsn = finance.BuildDatabaseURL(secret, dbHost, dbPort, dbName)
			log.Println("Successfully retrieved credentials from Secrets Manager")
		} else {
			// Fallback para DATABASE_URL hardcoded (retrocompatibilidade)
			log.Println("RDS_SECRET_NAME not set, using DATABASE_URL environment variable")
			dsn = mustGet("DATABASE_URL")
		}

		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(30 * time.Minute)
		if err := db.Ping(); err != nil {
			log.Fatalf("ping db: %v", err)
		}
		repo = finance.NewPostgresRepo(db)
		log.Println("storage=postgres connected")
	} else {
		repo = finance.NewMemoryRepo()
		log.Println("storage=memory")
	}

	svc := finance.NewService(repo)
	mux := httpapi.NewMux(svc)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustGet(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env var: %s", k)
	}
	return v
}