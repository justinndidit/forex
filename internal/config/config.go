package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Database DatabaseConfig `koanf:"database" validate:"required"`
	Server   ServerConfig   `koanf:"server" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	// Load DATABASE_* environment variables
	err := k.Load(env.ProviderWithValue("DATABASE_", ".", func(key, value string) (string, any) {
		// Transform DATABASE_HOST -> database.host
		cleanKey := strings.TrimPrefix(key, "DATABASE_")
		return "database." + strings.ToLower(cleanKey), value
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load database env variables")
	}

	// Load SERVER_* environment variables
	err = k.Load(env.ProviderWithValue("SERVER_", ".", func(key, value string) (string, any) {
		// Transform SERVER_PORT -> server.port
		cleanKey := strings.TrimPrefix(key, "SERVER_")
		return "server." + strings.ToLower(cleanKey), value
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load server env variables")
	}

	mainConfig := &Config{}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal main config")
	}

	validate := validator.New()

	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("config validation failed")
	}
	logger.Info().Msg("config validation passed")

	return mainConfig, nil
}
