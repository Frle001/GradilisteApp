package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ── Login ────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	ctx := c.Request.Context()
	db := GetDB()

	var (
		userID       string
		companyID    string
		employeeID   *string
		email        string
		passwordHash string
		role         string
		active       bool
	)

	err := db.QueryRow(ctx, `
		SELECT
			id::text,
			company_id::text,
			employee_id::text,
			email,
			password_hash,
			role,
			active
		FROM users
		WHERE email = $1
	`, req.Email).Scan(&userID, &companyID, &employeeID, &email, &passwordHash, &role, &active)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		log.Printf("login: db error for email %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is inactive"})
		return
	}

	if !CheckPassword(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Update last_login_at (best-effort, don't fail on error)
	if _, err := db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1::uuid`, userID); err != nil {
		log.Printf("login: failed to update last_login_at for user %s: %v", userID, err)
	}

	empID := ""
	if employeeID != nil {
		empID = *employeeID
	}

	token, err := GenerateAccessToken(userID, companyID, empID, role, email)
	if err != nil {
		log.Printf("login: token generation failed for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Async audit log — does not block the response
	go CreateAuditLog(context.Background(), db, AuditLogParams{
		CompanyID:  companyID,
		UserID:     &userID,
		Action:     "auth.login",
		EntityType: "user",
		EntityID:   &userID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	})

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"user": gin.H{
			"id":          userID,
			"company_id":  companyID,
			"employee_id": employeeID,
			"email":       email,
			"role":        role,
		},
	})
}

// ── Me ───────────────────────────────────────────────────────────────────────

func MeHandler(c *gin.Context) {
	authUser := GetAuthUser(c)
	ctx := c.Request.Context()
	db := GetDB()

	var (
		userID        string
		companyID     string
		employeeID    *string
		email         string
		role          string
		active        bool
		emailVerified bool
	)

	err := db.QueryRow(ctx, `
		SELECT
			id::text,
			company_id::text,
			employee_id::text,
			email,
			role,
			active,
			email_verified
		FROM users
		WHERE id = $1::uuid AND active = true
	`, authUser.UserID).Scan(&userID, &companyID, &employeeID, &email, &role, &active, &emailVerified)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or inactive"})
		return
	}
	if err != nil {
		log.Printf("me: db error for user %s: %v", authUser.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	userResp := gin.H{
		"id":             userID,
		"company_id":     companyID,
		"employee_id":    employeeID,
		"email":          email,
		"role":           role,
		"active":         active,
		"email_verified": emailVerified,
	}

	// Fetch linked employee if present
	var employeeResp interface{}
	if employeeID != nil && *employeeID != "" {
		var (
			empID        string
			empFirstName string
			empLastName  string
			empRole      string
		)
		empErr := db.QueryRow(ctx, `
			SELECT id::text, first_name, last_name, role
			FROM employees
			WHERE id = $1::uuid AND company_id = $2::uuid AND active = true
		`, *employeeID, companyID).Scan(&empID, &empFirstName, &empLastName, &empRole)

		if empErr == nil {
			employeeResp = gin.H{
				"id":         empID,
				"first_name": empFirstName,
				"last_name":  empLastName,
				"role":       empRole,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     userResp,
		"employee": employeeResp,
	})
}

// ── Logout ───────────────────────────────────────────────────────────────────

// LogoutHandler is stateless — JWT tokens are not invalidated server-side in Phase 3.
// The client is responsible for discarding the token.
func LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// ── Register (disabled) ──────────────────────────────────────────────────────

// RegisterHandler is intentionally disabled.
// This is a company-private app; accounts are created by administrators, not self-registered.
func RegisterHandler(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Public registration is disabled",
		"message": "Contact your administrator to create an account",
	})
}

// ── Protected test routes ────────────────────────────────────────────────────
// These exist only to verify that auth and role middleware work correctly.

func ProtectedMeHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{
		"user_id":     u.UserID,
		"company_id":  u.CompanyID,
		"employee_id": u.EmployeeID,
		"role":        u.Role,
		"email":       u.Email,
	})
}

func ProtectedDirectorEngineerHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "Access granted for direktor/inzenjer",
		"role":    u.Role,
	})
}

func ProtectedAdminHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "Access granted for direktor/inzenjer/administracija",
		"role":    u.Role,
	})
}

func ProtectedPoslovodaHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{
		"message": "Access granted for direktor/inzenjer/poslovoda",
		"role":    u.Role,
	})
}
