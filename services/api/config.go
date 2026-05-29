package main

import (
	"log"
	"os"
	"strconv"
)

// AppConfig holds parsed environment variables.
// Load once at startup via LoadConfig(); access via the global Config.
type AppConfig struct {
	Env          string
	JWTSecret    string
	JWTExpiresIn string
	BcryptCost   int
}

var Config *AppConfig

func LoadConfig() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "production"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if env == "development" {
			log.Println("WARNING: JWT_SECRET is not set. Using insecure dev default. Set JWT_SECRET in .env for any real use.")
			jwtSecret = "dev-secret-change-before-any-real-use"
		} else {
			log.Fatal("FATAL: JWT_SECRET environment variable is required in non-development environments")
		}
	}

	jwtExpiresIn := os.Getenv("JWT_EXPIRES_IN")
	if jwtExpiresIn == "" {
		jwtExpiresIn = "24h"
	}

	bcryptCost, _ := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if bcryptCost == 0 {
		bcryptCost = 12
	}

	Config = &AppConfig{
		Env:          env,
		JWTSecret:    jwtSecret,
		JWTExpiresIn: jwtExpiresIn,
		BcryptCost:   bcryptCost,
	}
}
