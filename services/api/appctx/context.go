package appctx

import "github.com/gin-gonic/gin"

const AuthUserKey = "auth_user"

// AuthContext holds the authenticated user's identity extracted from the JWT.
// Set by AuthRequired() in package main; read by handlers via GetAuthUser().
type AuthContext struct {
	UserID     string
	CompanyID  string
	EmployeeID string // empty when user has no linked employee record
	Role       string
	Email      string
}

// GetAuthUser returns the AuthContext injected by AuthRequired(), or nil.
func GetAuthUser(c *gin.Context) *AuthContext {
	val, exists := c.Get(AuthUserKey)
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
// All business queries must filter by this value — never accept company_id from the request.
func CompanyID(c *gin.Context) string {
	if u := GetAuthUser(c); u != nil {
		return u.CompanyID
	}
	return ""
}
