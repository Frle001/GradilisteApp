package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
)

// AuthRequired validates the Bearer token and injects AuthContext into the Gin context.
// Returns 401 for missing, invalid, or expired tokens.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required (Bearer token)",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		c.Set(appctx.AuthUserKey, &appctx.AuthContext{
			UserID:     claims.UserID,
			CompanyID:  claims.CompanyID,
			EmployeeID: claims.EmployeeID,
			Role:       claims.Role,
			Email:      claims.Email,
		})
		c.Next()
	}
}

// RequireRoles returns 403 if the authenticated user's role is not in the allowed list.
// Must be used after AuthRequired().
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		u := GetAuthUser(c)
		if u == nil || !allowed[u.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			return
		}
		c.Next()
	}
}

// GetAuthUser returns the AuthContext set by AuthRequired(), or nil.
func GetAuthUser(c *gin.Context) *appctx.AuthContext {
	return appctx.GetAuthUser(c)
}

// CompanyID returns the company_id from the authenticated user's JWT claims.
//
// IMPORTANT: All business queries must filter by this value.
// Never accept company_id from request body or query params — always derive it from the token.
// This is the primary multi-tenant isolation mechanism.
func CompanyID(c *gin.Context) string {
	return appctx.CompanyID(c)
}
