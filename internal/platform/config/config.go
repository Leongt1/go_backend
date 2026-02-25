package config

import (
	"log"
	"os"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Admin    AdminConfig
}

type DatabaseConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  string
	RefreshTTL string
}

type AdminConfig struct {
	Name     string
	Email    string
	Password string
}

func Load() *Config {
	cfg := &Config{
		Database: DatabaseConfig{
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  os.Getenv("ACCESS_TTL"),
			RefreshTTL: os.Getenv("REFRESH_TTL"),
		},
		Admin: AdminConfig{
			Name:     os.Getenv("ADMIN_NAME"),
			Email:    os.Getenv("ADMIN_EMAIL"),
			Password: os.Getenv("ADMIN_PASSWORD"),
		},
	}

	validate(cfg)
	return cfg
}

func validate(cfg *Config) {
	if cfg.Database.User == "" || cfg.Database.Host == "" || cfg.Database.Name == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if cfg.JWT.Secret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if cfg.JWT.AccessTTL == "" {
		log.Fatal("ACCESS_TTL is required")
	}

	if cfg.JWT.RefreshTTL == "" {
		log.Fatal("REFRESH_TTL is required")
	}

	if cfg.Admin.Email == "" {
		log.Fatal("ADMIN_EMAIL is required")
	}

	if cfg.Admin.Password == "" {
		log.Fatal("ADMIN_PASSWORD is required")
	}

	if cfg.Admin.Name == "" {
		log.Fatal("ADMIN_NAME is required")
	}
}
