package config

import (
	"log"
	"os"
)

type Config struct {
	Database        DatabaseConfig
	JWT             JWTConfig
	Admin           AdminConfig
	ResetPassword   ResetPasswordConfig
	Email           EmailConfig
	FrontendBaseURL string
	ServerPort      string
}

type DatabaseConfig struct {
	User           string
	Password       string
	Host           string
	Port           string
	Name           string
	SSLmode        string
	ChannelBinding string
}

type JWTConfig struct {
	Secret     string
	AccessTTL  string
	RefreshTTL string
}

type ResetPasswordConfig struct {
	ResetPasswordTTL string
}

type AdminConfig struct {
	Name     string
	Email    string
	Password string
}

type ResendConfig struct {
	APIKey string
	From   string
}

type SMTPConfig struct {
	MailTrapAPIKey string
	Host           string
	Port           string
	Username       string
	Password       string
	From           string
}

type EmailConfig struct {
	Resend ResendConfig
	SMTP   SMTPConfig
}

func Load() *Config {
	cfg := &Config{
		Database: DatabaseConfig{
			User:           os.Getenv("DB_USER"),
			Password:       os.Getenv("DB_PASSWORD"),
			Host:           os.Getenv("DB_HOST"),
			Port:           os.Getenv("DB_PORT"),
			Name:           os.Getenv("DB_NAME"),
			SSLmode:        os.Getenv("PGSSLMODE"),
			ChannelBinding: os.Getenv("PGCHANNELBINDING"),
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
		ResetPassword: ResetPasswordConfig{
			ResetPasswordTTL: os.Getenv("RESET_PASSWORD_TTL"),
		},
		Email: EmailConfig{
			Resend: ResendConfig{
				APIKey: os.Getenv("RESEND_API_KEY"),
				From:   os.Getenv("RESEND_FROM"),
			},
			SMTP: SMTPConfig{
				MailTrapAPIKey: os.Getenv("MAILTRAP_API_KEY"),
				Host:           os.Getenv("SMTP_HOST"),
				Port:           os.Getenv("SMTP_PORT"),
				Username:       os.Getenv("SMTP_USERNAME"),
				Password:       os.Getenv("SMTP_PASSWORD"),
				From:           os.Getenv("SMTP_FROM"),
			},
		},
		FrontendBaseURL: os.Getenv("FRONTEND_URL"),
		ServerPort:      os.Getenv("PORT"),
	}

	validate(cfg)
	return cfg
}

func validate(cfg *Config) {
	if cfg.Database.User == "" || cfg.Database.Host == "" || cfg.Database.Name == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if cfg.ServerPort == "" {
		log.Fatal("PORT is required")
	}

	if cfg.Database.SSLmode == "" {
		log.Fatal("SSLmode is required")
	}

	if cfg.Database.ChannelBinding == "" {
		log.Fatal("Channel Binding is required")
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

	if cfg.ResetPassword.ResetPasswordTTL == "" {
		log.Fatal("RESET_PASSWORD_TTL is required")
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

	if cfg.FrontendBaseURL == "" {
		log.Fatal("FRONTEND BASE URL is required")
	}
}
