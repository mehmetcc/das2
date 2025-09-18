package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type AppConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DbConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime time.Duration
}

type JWTConfig struct {
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	JWTIssuer   string
	JWTAudience string
	JWTAlg      string
	JWTKID      string
}

type CoookieConfig struct {
	CookieDomain   string
	CookieSecure   string
	CookieSamesite string
}

type Config struct {
	AppConfig    *AppConfig
	DbConfig     *DbConfig
	JWTConfig    *JWTConfig
	CookieConfig *CoookieConfig
}

func LoadConfig(logger *zap.Logger) (*Config, error) {
	if err := godotenv.Load("./../.env"); err != nil {
		logger.Error("failed to load .env file", zap.Error(err))
		return nil, err
	}

	/** db config */
	dsn := os.Getenv("POSTGRES_DSN")

	mocs := os.Getenv("DB_MAX_OPEN_CONNS")
	mics := os.Getenv("DB_MAX_IDLE_CONNS")
	mcls := os.Getenv("DB_CONN_MAX_LIFETIME")

	maxOpenConns, err := strconv.Atoi(mocs)
	if err != nil {
		return nil, err
	}
	maxIdleConns, err := strconv.Atoi(mics)
	if err != nil {
		return nil, err
	}
	maxConnLifetimeDuration, err := time.ParseDuration(mcls)
	if err != nil {
		return nil, err
	}

	dbConfig := &DbConfig{
		DSN:             dsn,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		MaxConnLifetime: maxConnLifetimeDuration,
	}

	/** app config */
	port := os.Getenv("APP_PORT")

	rts := os.Getenv("APP_READ_TIMEOUT")
	wts := os.Getenv("APP_WRITE_TIMEOUT")
	its := os.Getenv("APP_IDLE_TIMEOUT")

	readTimeoutDuration, err := time.ParseDuration(rts)
	if err != nil {
		return nil, err
	}
	writeTimeoutDuration, err := time.ParseDuration(wts)
	if err != nil {
		return nil, err
	}
	idleTimeoutDuration, err := time.ParseDuration(its)
	if err != nil {
		return nil, err
	}

	appConfig := &AppConfig{
		Port:         port,
		ReadTimeout:  readTimeoutDuration,
		WriteTimeout: writeTimeoutDuration,
		IdleTimeout:  idleTimeoutDuration,
	}

	/** jwt config */
	attls := os.Getenv("ACCESS_TTL")
	accessTTL, err := time.ParseDuration(attls)
	if err != nil {
		return nil, err
	}
	rttls := os.Getenv("REFRESH_TTL")
	refreshTTL, err := time.ParseDuration(rttls)
	if err != nil {
		return nil, err
	}
	jwtConfig := &JWTConfig{
		AccessTTL:   accessTTL,
		RefreshTTL:  refreshTTL,
		JWTIssuer:   os.Getenv("JWT_ISSUER"),
		JWTAudience: os.Getenv("JWT_AUDIENCE"),
		JWTAlg:      os.Getenv("JWT_ALG"),
		JWTKID:      os.Getenv("JWT_KID"),
	}

	/** cookie config */
	cookieConfig := &CoookieConfig{
		CookieDomain:   os.Getenv("COOKIE_DOMAIN"),
		CookieSecure:   os.Getenv("COOKIE_SECURE"),
		CookieSamesite: os.Getenv("COOKIE_SAMESITE"),
	}

	return &Config{
		DbConfig:     dbConfig,
		AppConfig:    appConfig,
		JWTConfig:    jwtConfig,
		CookieConfig: cookieConfig,
	}, nil
}
