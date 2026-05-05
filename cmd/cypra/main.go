// Package main provides the Cypra CLI entrypoint.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/watzon/cypra/dashboard"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("cypra %s (%s)\n", version, commit)
		return
	}
	if err := run(os.Args[1:]); err != nil {
		var exitErr exitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.error)
			os.Exit(exitErr.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return exitError{code: 2, error: "cypra: command required"}
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "migrate":
		return runMigrate()
	default:
		return exitError{code: 2, error: "cypra: unknown command " + args[0]}
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	skipMigrate := fs.Bool("skip-migrate", false, "skip boot-time migration check")
	if err := fs.Parse(args); err != nil {
		return exitError{code: 2, error: err.Error()}
	}
	publicBaseURL := envDefault("PUBLIC_BASE_URL", "https://cypra.localhost")
	if err := validatePublicBaseURL(publicBaseURL); err != nil {
		return err
	}
	dbConn, err := openDB(envDefault("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	if !*skipMigrate {
		pending, err := migrate.Pending(context.Background(), dbConn, "")
		if err != nil {
			return err
		}
		if pending {
			return exitError{code: 4, error: "Pending migrations. Run `cypra migrate`."}
		}
	}
	gormDB, err := gorm.Open(postgres.Open(envDefault("DATABASE_URL", "")), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open gorm db: %w", err)
	}
	server, err := httpserver.New(httpserver.Options{DB: dbConn, TenantDB: db.NewTenantScopedDB(gormDB), PublicBaseURL: publicBaseURL, Version: version, Commit: commit, DevOpenAPI: os.Getenv("LOG_LEVEL") == "debug", KEKLoaded: masterKeyConfigured()})
	if err != nil {
		return err
	}
	httpServer := &http.Server{Addr: envDefault("LISTEN_ADDR", ":8080"), Handler: server.Router(), ReadHeaderTimeout: 5 * time.Second}
	return httpServer.ListenAndServe()
}

func runMigrate() error {
	dsn := os.Getenv("MIGRATE_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
		fmt.Fprintln(os.Stderr, "warning: MIGRATE_DATABASE_URL unset; falling back to DATABASE_URL")
	}
	dbConn, err := openDB(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()
	return migrate.Apply(context.Background(), dbConn, "")
}

func openDB(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return dbConn, nil
}

func validatePublicBaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("PUBLIC_BASE_URL must be a valid URL")
	}
	if parsed.Scheme != "https" && os.Getenv("CYPRA_DEV_INSECURE_HTTP") != "true" {
		return fmt.Errorf("PUBLIC_BASE_URL must be HTTPS outside dev")
	}
	return nil
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func masterKeyConfigured() bool {
	return strings.TrimSpace(os.Getenv("MASTER_KEY")) != "" || strings.TrimSpace(os.Getenv("MASTER_KEY_FILE")) != ""
}

type exitError struct {
	code  int
	error string
}

func (e exitError) Error() string {
	return e.error
}
