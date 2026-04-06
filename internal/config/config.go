// Package config loads and validates environment variables at startup.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost, DBPort, DBUser, DBPassword, DBName, DBSSLMode string
	JWTSecret                                             string
	JWTAccessExpiry, JWTRefreshExpiry                     time.Duration
	ServerPort, AppEnv                                    string
	RateLimitRPM                                          int
	GINMode                                               string
}

var C Config

func Load() error {
	_ = godotenv.Load()
	if os.Getenv("JWT_SECRET") == "" || os.Getenv("DB_HOST") == "" {
		return errors.New("JWT_SECRET and DB_HOST are required")
	}
	ae, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	re, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	rpmStr := getEnv("RATE_LIMIT_RPM", "100")
	rpm, err := strconv.Atoi(rpmStr)
	ginMode := getEnv("GIN_MODE", "release")
	if err != nil || rpm <= 0 {
		rpm = 100
	}
	C = Config{
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTAccessExpiry:  ae,
		JWTRefreshExpiry: re,
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		AppEnv:           getEnv("APP_ENV", "development"),
		RateLimitRPM:     rpm,
		GINMode:          ginMode,
	}
	return nil
}

func getEnv(k, fb string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fb
}
