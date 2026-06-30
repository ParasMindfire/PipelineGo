package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	_ "pipeline/apps/server/docs"

	"pipeline/apps/server/controller"
	"pipeline/apps/server/repository"
	"pipeline/apps/server/routes"
	"pipeline/apps/server/service"
	"pipeline/packages/shared/config"
	"pipeline/packages/shared/logger"
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
	log := logger.New("info")

	if err := godotenv.Load(); err != nil {
		log.Warn("no .env file found, using environment variables")
	}

	cfg := config.DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   getEnv("DB_NAME", "pipeline_db"),
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		log.Error("failed to connect to database", map[string]interface{}{"err": err.Error()})
		return 1
	}
	defer db.Close()

	repo := repository.NewPipelineRepository(db)
	runner := pipelines.NewRunner(repo)
	svc := service.NewPipelineService(repo, runner)
	ctrl := controller.NewPipelineController(svc)
	router := routes.NewRouter(ctrl)

	srv := &http.Server{
		Addr:         ":" + getEnv("PORT", "8080"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	serveErrCh := make(chan error, 1)
	go func() {
		log.Info("pipeline server starting", map[string]interface{}{"addr": srv.Addr})
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
			log.Error("server failed", map[string]interface{}{"err": err.Error()})
			return 1
		}
	case <-quit:
		log.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", map[string]interface{}{"err": err.Error()})
			return 1
		}
		log.Info("stopped")
	}
	return 0
}
