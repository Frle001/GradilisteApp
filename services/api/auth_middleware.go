package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const authUserKey = "auth_user"

// AuthContext holds the authenticated user's identity extracted from the JWT.
// Set by AuthRequired(); read by GetAuthUser() and CompanyID().
type AuthContext struct {
	UserID     string
	CompanyID  string
	EmployeeID string // empty when user has no linked employee record
	Role       string
	Email      string
}

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

		c.Set(authUserKey, &AuthContext{
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
func GetAuthUser(c *gin.Context) *AuthContext {
	val, exists := c.Get(authUserKey)
	if !exists {
		return nil
	}
	u, ok := val.(*AuthContext)
	if !ok {
		return nil
	}
	return u
}

// CompanyID returns the company_id from the authenticated user's JWT claims.
//
// IMPORTANT: All business queries must filter by this value.
// Never accept company_id from request body or query params — always derive it from the token.
// This is the primary multi-tenant isolation mechanism.
func CompanyID(c *gin.Context) string {
	if u := GetAuthUser(c); u != nil {
		return u.CompanyID
	}
	return ""
}
