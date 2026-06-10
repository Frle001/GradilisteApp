package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// errResp sends the standard error envelope:
//
//	{ "error": { "code": "...", "message": "..." } }
//
// All new handlers should use this. Existing business handlers in the handlers/
// sub-package keep their current format and can migrate gradually.
func errResp(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// Common shorthand wrappers used in auth handlers.
func errBadRequest(c *gin.Context, message string) {
	errResp(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

func errUnauthorized(c *gin.Context, message string) {
	errResp(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func errInternal(c *gin.Context) {
	errResp(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}

func errTooManyRequests(c *gin.Context) {
	errResp(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests — please try again later")
}
