package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	_ "pipeline/apps/server/docs"

	"pipeline/apps/server/controller"
	"pipeline/apps/server/repository"
	"pipeline/apps/server/routes"
	"pipeline/apps/server/service"
	"pipeline/packages/shared/config"
	"pipeline/packages/shared/pipelines"
)

// @title Pipeline Builder API
// @version 1.0
// @description Data ingestion, validation, transformation, and aggregation pipeline service.
// @BasePath /

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	os.Exit(run())
}

// run holds the actual startup logic so deferred cleanup (db.Close) always
// executes before the process exits — os.Exit skips deferred calls.
func run() int {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	cfg := config.DBConfig{
		Host:     getEnv("DB_HOST", defaultDBHost),
		Port:     getEnv("DB_PORT", defaultDBPort),
		User:     getEnv("DB_USER", defaultDBUser),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   getEnv("DB_NAME", defaultDBName),
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Println("API_KEY is not set — refusing to start with mutating endpoints unprotected")
		return 1
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		log.Printf("failed to connect to database: %v", err)
		return 1
	}
	defer db.Close()

	repo := repository.NewPipelineRepository(db)
	runner := pipelines.NewRunner(repo)
	svc := service.NewPipelineService(repo, runner)
	ctrl := controller.NewPipelineController(svc)
	router := routes.NewRouter(ctrl, apiKey)

	srv := &http.Server{
		Addr:         ":" + getEnv("PORT", defaultPort),
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
	}

	serveErrCh := make(chan error, 1)
	go func() {
		log.Printf("pipeline server starting addr=%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErrCh <- err
			return
		}
		serveErrCh <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErrCh:
		if err != nil {
			log.Printf("server failed: %v", err)
			return 1
		}
	case <-quit:
		log.Println("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			return 1
		}
		log.Println("stopped")
	}
	return 0
}
