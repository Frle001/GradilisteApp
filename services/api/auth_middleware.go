package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/repositories"
)

// authStateChecker is satisfied by *repositories.UserRepository.
type authStateChecker interface {
	GetAuthState(ctx context.Context, userID string) (*repositories.AuthState, error)
}

// checkAuthVersionAllowed returns true when the JWT's auth_version is current.
// A JWT with AuthVersion == 0 is a legacy token (issued before this field was added);
// it is treated as version 1 and accepted only when the DB version is also 1.
func checkAuthVersionAllowed(jwtVersion, dbVersion int) bool {
	if jwtVersion == 0 {
		jwtVersion = 1
	}
	return jwtVersion == dbVersion
}

// AuthRequired validates the Bearer token and then verifies the user is still
// active and that the token's auth_version matches the DB. Returns 401 for any
// failure so the client knows to re-authenticate.
func AuthRequired(repo authStateChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		state, err := repo.GetAuthState(c.Request.Context(), claims.UserID)
		if err != nil || !state.Active || !checkAuthVersionAllowed(claims.AuthVersion, state.AuthVersion) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
