package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Logger   LoggerConfig
	Postgres PostgresConfig
	JWT      JWTConfig
	Kafka    KafkaConfig
}

type ServerConfig struct {
	AppEnv   string
	GRPCPort string
}

type LoggerConfig struct {
	Level             string
	Encoding          string
	DisableCaller     bool
	DisableStacktrace bool
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type JWTConfig struct {
	SecretKey string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string // orders.events
	GroupID string // customer-service
}

func LoadEnv() *Config {
	return &Config{
		Server: ServerConfig{
			AppEnv:   getEnv("APP_ENV", "dev"),
			GRPCPort: getEnv("GRPC_PORT", ":8084"), // Port 8084 for Customer Service
		},
		Logger: LoggerConfig{
			Level:             getEnv("LOGGER_LEVEL", "debug"),
			Encoding:          getEnv("LOGGER_ENCODING", "console"),
			DisableCaller:     getEnvBool("LOGGER_DISABLE_CALLER", false),
			DisableStacktrace: getEnvBool("LOGGER_DISABLE_STACKTRACE", true),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5433"),
			User:            getEnv("POSTGRES_USER", "omnipos"),
			Password:        getEnv("POSTGRES_PASSWORD", "omnipos"),
			DBName:          getEnv("POSTGRES_DB", "omnipos_customer_db"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("POSTGRES_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvInt("POSTGRES_CONN_MAX_LIFETIME", 300)) * time.Second,
			ConnMaxIdleTime: time.Duration(getEnvInt("POSTGRES_CONN_MAX_IDLE_TIME", 60)) * time.Second,
		},
		JWT: JWTConfig{
			SecretKey: getEnv("JWT_SECRET_KEY", "your-secret-key"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:29092")},
			Topic:   getEnv("KAFKA_TOPIC", "orders.events"),
			GroupID: getEnv("KAFKA_GROUP_ID", "customer-service"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
