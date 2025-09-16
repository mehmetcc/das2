package main

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/mehmetcc/das2/internal/database"
	"go.uber.org/zap"
)

func main() {
	// init logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	// load dotenv file
	err = godotenv.Load("../.env")
	if err != nil {
		logger.Error("failed to load .env file", zap.Error(err))
	}

	// load database
	db, err := database.Init()
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// run migrations
	database.SetMigrationLogger(logger)
	err = database.Migrate(context.Background(), db)
	if err != nil {
		logger.Error("failed to migrate database", zap.Error(err))
	}

	// happy hacking bitches
	logger.Info("application started")
}
