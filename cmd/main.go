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
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/joho/godotenv"
	"github.com/mehmetcc/das2/internal/auth"
	"github.com/mehmetcc/das2/internal/database"
	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/internal/session"
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
	router.Use(httprate.Limit(
		2,              // requests
		10*time.Second, // per duration
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	))
	router.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("It is alive!"))
	})

	// initialize components
	personRepo := person.NewPersonRepo(db, logger)

	sessionRepo := session.NewSessionRepo(db, logger)

	authService := auth.NewAuthenticationService(personRepo, sessionRepo, logger)
	authHandler := auth.NewAuthenticationHandler(authService, logger)
	router.Mount("/auth", authHandler.Routes())

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
