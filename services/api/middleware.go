package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const ctxKeyRequestID = "request_id"

// RequestIDMiddleware attaches a unique ID to every request (reads X-Request-ID
// header or generates one) and echoes it back in the response header.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			buf := make([]byte, 8)
			_, _ = rand.Read(buf)
			id = hex.EncodeToString(buf)
		}
		c.Set(ctxKeyRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// StructuredLogMiddleware logs each request via slog: method, path, status,
// duration, and request ID. Replaces gin.Logger().
func StructuredLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()
		reqID, _ := c.Get(ctxKeyRequestID)

		attrs := []any{
			slog.String("request_id", strVal(reqID)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.ClientIP()),
		}
		if errs := c.Errors.String(); errs != "" {
			attrs = append(attrs, slog.String("errors", errs))
		}

		switch {
		case status >= 500:
			slog.Error("request", attrs...)
		case status >= 400:
			slog.Warn("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// BodySizeLimitMiddleware rejects bodies larger than maxBytes before Gin reads them.
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// CORSMiddleware sets Access-Control-* headers from Config.CORSAllowedOrigins.
// In development (empty origins list) all origins are reflected.
// In production only listed origins are accepted; other origins get a 403 on preflight.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := isCORSAllowed(origin)

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Request-ID, X-CSRF-Token, Accept, Origin, Cache-Control")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}

		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}

		c.Next()
	}
}

func isCORSAllowed(origin string) bool {
	if origin == "" {
		return true // non-browser request
	}
	if len(Config.CORSAllowedOrigins) == 0 {
		return true // development: allow all
	}
	for _, allowed := range Config.CORSAllowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}
