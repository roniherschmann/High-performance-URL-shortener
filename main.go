package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/yourname/go-shorty/internal/config"
	"github.com/yourname/go-shorty/internal/core"
	httpapi "github.com/yourname/go-shorty/internal/http"
	"github.com/yourname/go-shorty/internal/store"
)

func main() {
	// Fast JSON logs by default; pretty if running in a TTY/dev
	if isatty() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		zerolog.TimeFieldFormat = time.RFC3339
	}

	cfg := config.Load()

	var dsnFlag string
	flag.StringVar(&dsnFlag, "dsn", "", "SQLite DSN (overrides env DB_DSN)")
	flag.Parse()
	if dsnFlag != "" {
		cfg.DBDSN = dsnFlag
	}

	db, err := sql.Open("sqlite3", cfg.DBDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("open sqlite")
	}
	defer db.Close()

	// Connection pool tuning
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Migrate schema
	if err := store.Migrate(db); err != nil {
		log.Fatal().Err(err).Msg("migrate schema")
	}

	// Create store + service
	sqlStore := store.NewSQLite(db)
	svc := core.NewService(sqlStore)

	// Start async click ingester
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.RunClickIngester(ctx)

	// Prewarm cache
	if n := cfg.CachePrewarm; n > 0 {
		if err := svc.PrewarmCache(n); err != nil {
			log.Warn().Err(err).Msg("cache prewarm")
		}
	}

	// HTTP server
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           httpapi.NewRouter(cfg, svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Info().Int("port", cfg.Port).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutdown signal")
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server shutdown")
	}
	log.Info().Msg("bye")
}

func isatty() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
