package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DB_DSN     string
	JWT_SECRET string
	PORT       string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using OS env variables if any")
	}

	cfg := &Config{
		DB_DSN:     os.Getenv("DB_DSN"),
		JWT_SECRET: os.Getenv("JWT_SECRET"),
		PORT:       os.Getenv("PORT"),
	}

	if cfg.PORT == "" {
		cfg.PORT = "5000"
	}

	if cfg.DB_DSN == "" {
		log.Fatal("DB_DSN is required in environment variables")
	}
	if cfg.JWT_SECRET == "" {
		log.Fatal("JWT_SECRET is required in environment variables")
	}

	return cfg
}
