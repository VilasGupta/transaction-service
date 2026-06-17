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

	"github.com/VilasGupta/transaction-service/internal/handler"
	"github.com/VilasGupta/transaction-service/internal/migrate"
	"github.com/VilasGupta/transaction-service/internal/store"
)

func main() {
	cfg := loadConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.dbUser, cfg.dbPass, cfg.dbHost, cfg.dbPort, cfg.dbName)

	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := migrate.Run(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	s := store.NewMySQLStore(db)
	h := handler.New(s)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
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

	go func() {
		log.Printf("server listening on :%s", cfg.serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

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

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

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
