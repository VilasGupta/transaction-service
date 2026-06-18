package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/VilasGupta/transaction-service/docs"
	"github.com/VilasGupta/transaction-service/internal/handler"
	"github.com/VilasGupta/transaction-service/internal/migrate"
	"github.com/VilasGupta/transaction-service/internal/store"
)

// @title Transaction Service API
// @version 1.0
// @description REST API for managing cardholder accounts and financial transactions.

// @host localhost:3000
// @BasePath /
func main() {
	// Load configuration from environment variables
	cfg := loadConfig()

	// Build MySQL DSN; parseTime=true so driver scans DATETIME into time.Time
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.dbUser, cfg.dbPass, cfg.dbHost, cfg.dbPort, cfg.dbName)

	// Connect to MySQL with retry
	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run schema migrations on startup
	if err := migrate.Run(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Wire up store and handlers
	s := store.NewMySQLStore(db)
	h := handler.New(s)

	// Register routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)
	mux.HandleFunc("POST /accounts", h.CreateAccount)
	mux.HandleFunc("GET /accounts/{accountId}", h.GetAccount)
	mux.HandleFunc("POST /transactions", h.CreateTransaction)

	srv := &http.Server{
		Addr:         ":" + cfg.serverPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can listen for shutdown signals
	go func() {
		log.Printf("server listening on :%s", cfg.serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM is received
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown: allow in-flight requests up to 15s to complete
	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}

type config struct {
	dbHost     string
	dbPort     string
	dbUser     string
	dbPass     string
	dbName     string
	serverPort string
}

// loadConfig reads server and database settings from environment variables, falling back to defaults.
func loadConfig() config {
	return config{
		dbHost:     envOrDefault("DB_HOST", "localhost"),
		dbPort:     envOrDefault("DB_PORT", "3306"),
		dbUser:     envOrDefault("DB_USER", "root"),
		dbPass:     envOrDefault("DB_PASSWORD", ""),
		dbName:     envOrDefault("DB_NAME", "transactions"),
		serverPort: envOrDefault("SERVER_PORT", "8080"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// openDB opens a MySQL connection pool and retries Ping up to 10 times with incremental backoff.
func openDB(dsn string) (*sql.DB, error) {
	// sql.Open only validates the DSN; it doesn't establish a connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Configure connection pool limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Retry Ping with incremental backoff (useful when waiting for Docker MySQL to start)
	for i := range 10 {
		if err = db.Ping(); err == nil {
			return db, nil
		}

		wait := time.Duration(i+1) * time.Second
		log.Printf("db not ready, retrying in %v... (%d/10)", wait, i+1)
		time.Sleep(wait)
	}

	db.Close()
	return nil, fmt.Errorf("database not reachable after 10 attempts: %w", err)
}
