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
	"github.com/mehmetcc/das2/internal/auth"
	"github.com/mehmetcc/das2/internal/config"
	"github.com/mehmetcc/das2/internal/database"
	"github.com/mehmetcc/das2/internal/person"
	"github.com/mehmetcc/das2/internal/session"
	"github.com/mehmetcc/das2/internal/token"
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

	// load config
	cfg, err := config.LoadConfig(logger)
	if err != nil {
		logger.Panic("err loading config", zap.Error(err))
	}

	// load database
	db, err := database.Init(cfg)
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
	// I took this directly from cors middleware github readme
	// TODO: investigate if this is suitable for this app
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
	refreshRepo := token.NewRefreshTokenRepo(db, logger)

	tokenService := token.NewTokenService(logger, refreshRepo, cfg.JWTConfig)
	authService := auth.NewAuthenticationService(personRepo, sessionRepo, tokenService, logger)

	authHandler := auth.NewAuthenticationHandler(authService, logger)
	router.Mount("/auth", authHandler.Routes())

	// generate and run a server
	server := &http.Server{
		Addr:         cfg.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  cfg.AppConfig.ReadTimeout,
		WriteTimeout: cfg.AppConfig.WriteTimeout,
		IdleTimeout:  cfg.AppConfig.IdleTimeout,
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
