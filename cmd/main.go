package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/mehmetcc/das2/internal/database"
	"go.uber.org/zap"
	"moul.io/chizap"
)

func main() {
	// init logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// load dotenv file
	if err := godotenv.Load("../.env"); err != nil {
		logger.Error("failed to load .env file", zap.Error(err))
	}

	// load database
	db, err := database.Init()
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// add root context
	// we will use this for graceful shutdown, and tracking whether migrations failed
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// run migrations
	migrateCtx, cancelMig := context.WithTimeout(appCtx, 60*time.Second)
	defer cancelMig()
	if err := database.Migrate(migrateCtx, db); err != nil {
		logger.Fatal("failed to migrate database", zap.Error(err))
	}

	// register handler
	router := chi.NewRouter()
	router.Use(chizap.New(logger, &chizap.Opts{
		WithReferer:   true,
		WithUserAgent: true,
	}))
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("It is alive!"))
	})

	// generate and run a server
	// TODO: configure
	server := &http.Server{
		Addr:         ":666",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	logger.Info("started http server", zap.String("addr", server.Addr))

	select {
	case <-appCtx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("http server crashed", zap.Error(err))
		}
	}

	// graceful shutdown on SIGINT/SIGTERM
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(appCtx), 10*time.Second)
	defer cancel()

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("goodbye")
}
