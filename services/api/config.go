package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// AppConfig holds all runtime configuration derived from environment variables.
// Loaded once at startup via LoadConfig(); accessed globally via Config.
type AppConfig struct {
	Env                 string
	ServerPort          string
	AppVersion          string // set via APP_VERSION env var or build ldflags
	JWTSecret           string
	JWTExpiresIn        string
	BcryptCost          int
	CORSAllowedOrigins  []string // empty = allow all (dev only)
	UploadStorageDriver string   // "local" | "s3"
	LocalUploadDir      string
	MaxUploadSizeMB     int64
	LogLevel            string
	// Cookie settings — override with COOKIE_SECURE and COOKIE_SAME_SITE env vars.
	// For cross-origin HTTPS (Vercel frontend + Render API): COOKIE_SECURE=true, COOKIE_SAME_SITE=none
	CookieSecure   bool
	CookieSameSite http.SameSite
	// S3 / Cloudflare R2 storage settings — used when UPLOAD_STORAGE_DRIVER=s3
	S3Endpoint      string
	S3Bucket        string
	S3AccessKeyID   string
	S3SecretKey     string
	S3Region        string
	S3UsePathStyle  bool   // must be true for Cloudflare R2
	S3PublicBaseURL string // optional: direct public URL prefix for pre-signed URLs
}

var Config *AppConfig

func LoadConfig() {
	// APP_ENV is canonical; fall back to ENV for backward compatibility.
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "production"
	}

	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "dev"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if env == "development" {
			log.Println("WARNING: JWT_SECRET not set — using insecure dev placeholder. Never use this outside local development.")
			jwtSecret = "dev-secret-change-before-any-real-use"
		} else {
			log.Fatal("FATAL: JWT_SECRET is required in non-development environments")
		}
	}

	jwtExpiresIn := os.Getenv("JWT_EXPIRES_IN")
	if jwtExpiresIn == "" {
		jwtExpiresIn = "15m"
	}

	bcryptCost, _ := strconv.Atoi(os.Getenv("BCRYPT_COST"))
	if bcryptCost == 0 {
		bcryptCost = 12
	}

	// CORS_ALLOWED_ORIGINS: comma-separated list of allowed origins.
	// Required in production — omitting it will prevent the server from starting.
	corsRaw := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsOrigins []string
	for _, o := range strings.Split(corsRaw, ",") {
		if t := strings.TrimSpace(o); t != "" {
			corsOrigins = append(corsOrigins, t)
		}
	}
	if len(corsOrigins) == 0 && env != "development" {
		log.Fatal("FATAL: CORS_ALLOWED_ORIGINS must be set in non-development environments (e.g. https://app.example.com)")
	}

	uploadDriver := os.Getenv("UPLOAD_STORAGE_DRIVER")
	if uploadDriver == "" {
		uploadDriver = "local"
	}
	if uploadDriver != "local" && uploadDriver != "s3" {
		log.Fatalf("FATAL: UPLOAD_STORAGE_DRIVER must be 'local' or 's3', got %q", uploadDriver)
	}

	// Warn when local storage is used in production — container filesystem is ephemeral.
	if uploadDriver == "local" && env == "production" {
		log.Println("WARNING: UPLOAD_STORAGE_DRIVER=local in production. Receipt files will be lost on container restart. Set UPLOAD_STORAGE_DRIVER=s3 for persistent storage.")
	}

	localUploadDir := os.Getenv("LOCAL_UPLOAD_DIR")
	if localUploadDir == "" {
		localUploadDir = "./uploads"
	}

	maxUploadMB, _ := strconv.ParseInt(os.Getenv("MAX_UPLOAD_SIZE_MB"), 10, 64)
	if maxUploadMB <= 0 {
		maxUploadMB = 10
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		if env == "development" {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}

	// SERVER_PORT: prefer SERVER_PORT, fall back to PORT for platform compatibility.
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = os.Getenv("PORT")
	}
	if serverPort == "" {
		serverPort = "8080"
	}

	// COOKIE_SECURE — defaults to true outside development; override with "true"/"false".
	cookieSecure := env != "development"
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cookieSecure = parsed
		}
	}

	// COOKIE_SAME_SITE — "lax" | "none" | "strict".
	// For cross-origin HTTPS (Vercel frontend + Render API, or Cloudflare Tunnel): set to "none".
	cookieSameSite := http.SameSiteLaxMode
	switch strings.ToLower(os.Getenv("COOKIE_SAME_SITE")) {
	case "none":
		cookieSameSite = http.SameSiteNoneMode
	case "strict":
		cookieSameSite = http.SameSiteStrictMode
	case "lax":
		cookieSameSite = http.SameSiteLaxMode
	}

	// ── S3 / Cloudflare R2 configuration ─────────────────────────────────────
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3Bucket := os.Getenv("S3_BUCKET")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY_ID")
	s3SecretKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		s3Region = "auto" // Cloudflare R2 default
	}
	s3UsePathStyle := false
	if v := os.Getenv("S3_USE_PATH_STYLE"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			s3UsePathStyle = parsed
		}
	}
	s3PublicBaseURL := os.Getenv("S3_PUBLIC_BASE_URL")

	if uploadDriver == "s3" {
		if s3Bucket == "" {
			log.Fatal("FATAL: S3_BUCKET is required when UPLOAD_STORAGE_DRIVER=s3")
		}
		if s3AccessKey == "" || s3SecretKey == "" {
			log.Fatal("FATAL: S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are required when UPLOAD_STORAGE_DRIVER=s3")
		}
	}

	Config = &AppConfig{
		Env:                 env,
		ServerPort:          serverPort,
		AppVersion:          appVersion,
		JWTSecret:           jwtSecret,
		JWTExpiresIn:        jwtExpiresIn,
		BcryptCost:          bcryptCost,
		CORSAllowedOrigins:  corsOrigins,
		UploadStorageDriver: uploadDriver,
		LocalUploadDir:      localUploadDir,
		MaxUploadSizeMB:     maxUploadMB,
		LogLevel:            logLevel,
		CookieSecure:        cookieSecure,
		CookieSameSite:      cookieSameSite,
		S3Endpoint:          s3Endpoint,
		S3Bucket:            s3Bucket,
		S3AccessKeyID:       s3AccessKey,
		S3SecretKey:         s3SecretKey,
		S3Region:            s3Region,
		S3UsePathStyle:      s3UsePathStyle,
		S3PublicBaseURL:     s3PublicBaseURL,
	}

	configureSlog(logLevel)

	slog.Info("config loaded", "env", env, "version", appVersion, "storage_driver", uploadDriver)
}

func configureSlog(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
