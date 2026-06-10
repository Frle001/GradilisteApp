package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const (
	refreshCookieName = "refresh_token"
	refreshTokenTTL   = 30 * 24 * time.Hour // 30 days
)

// ── Login ─────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	db := GetDB()

	var (
		userID             string
		companyID          string
		employeeID         *string
		email              string
		passwordHash       string
		role               string
		active             bool
		mustChangePassword bool
	)

	err := db.QueryRow(ctx, `
		SELECT
			id::text,
			company_id::text,
			employee_id::text,
			email,
			password_hash,
			role,
			active,
			must_change_password
		FROM users
		WHERE email = $1
	`, req.Email).Scan(&userID, &companyID, &employeeID, &email, &passwordHash, &role, &active, &mustChangePassword)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		slog.Error("login: db error", "email", req.Email, "error", err)
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

	// Update last_login_at (best-effort)
	if _, err := db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1::uuid`, userID); err != nil {
		slog.Warn("login: failed to update last_login_at", "user_id", userID, "error", err)
	}

	empID := ""
	if employeeID != nil {
		empID = *employeeID
	}

	accessToken, err := GenerateAccessToken(userID, companyID, empID, role, email)
	if err != nil {
		slog.Error("login: access token generation failed", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Issue refresh token and store its hash in the DB.
	rawRefresh, refreshHash, err := GenerateRefreshToken()
	if err != nil {
		slog.Error("login: refresh token generation failed", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	expiresAt := time.Now().Add(refreshTokenTTL)
	if _, dbErr := db.Exec(ctx, `
		INSERT INTO refresh_tokens (company_id, user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
	`, companyID, userID, refreshHash, expiresAt, c.GetHeader("User-Agent"), c.ClientIP()); dbErr != nil {
		// Non-fatal: access token still works. Log and continue.
		slog.Error("login: failed to persist refresh token", "user_id", userID, "error", dbErr)
	} else {
		setRefreshCookie(c, rawRefresh)
	}

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
		"access_token": accessToken,
		"user": gin.H{
			"id":          userID,
			"company_id":  companyID,
			"employee_id": employeeID,
			"email":       email,
			"role":        role,
		},
		"mustChangePassword": mustChangePassword,
	})
}

// ── Refresh ───────────────────────────────────────────────────────────────────

// RefreshHandler validates the HTTP-only refresh cookie, rotates the token,
// and issues a new short-lived access token.
func RefreshHandler(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err != nil || rawToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No refresh token"})
		return
	}

	tokenHash := HashRefreshToken(rawToken)
	ctx := c.Request.Context()
	db := GetDB()

	var (
		tokenID   string
		companyID string
		userID    string
		expiresAt time.Time
		revokedAt *time.Time
	)
	err = db.QueryRow(ctx, `
		SELECT id::text, company_id::text, user_id::text, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&tokenID, &companyID, &userID, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}
	if err != nil {
		slog.Error("refresh: db lookup failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if revokedAt != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has been revoked"})
		return
	}
	if time.Now().After(expiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has expired"})
		return
	}

	// Load current user
	var (
		employeeID         *string
		email              string
		role               string
		active             bool
		mustChangePassword bool
	)
	err = db.QueryRow(ctx, `
		SELECT employee_id::text, email, role, active, must_change_password
		FROM users
		WHERE id = $1::uuid AND company_id = $2::uuid
	`, userID, companyID).Scan(&employeeID, &email, &role, &active, &mustChangePassword)
	if errors.Is(err, pgx.ErrNoRows) || !active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or inactive"})
		return
	}
	if err != nil {
		slog.Error("refresh: user lookup failed", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	empID := ""
	if employeeID != nil {
		empID = *employeeID
	}

	// Rotate: revoke old token, issue new one.
	if _, dbErr := db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1::uuid`, tokenID,
	); dbErr != nil {
		slog.Error("refresh: failed to revoke old token", "token_id", tokenID, "error", dbErr)
	}

	newRaw, newHash, err := GenerateRefreshToken()
	if err != nil {
		slog.Error("refresh: new token generation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	newExpiry := time.Now().Add(refreshTokenTTL)
	if _, dbErr := db.Exec(ctx, `
		INSERT INTO refresh_tokens (company_id, user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
	`, companyID, userID, newHash, newExpiry, c.GetHeader("User-Agent"), c.ClientIP()); dbErr != nil {
		slog.Error("refresh: failed to persist new token", "error", dbErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	accessToken, err := GenerateAccessToken(userID, companyID, empID, role, email)
	if err != nil {
		slog.Error("refresh: access token generation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	setRefreshCookie(c, newRaw)
	c.JSON(http.StatusOK, gin.H{
		"access_token":       accessToken,
		"mustChangePassword": mustChangePassword,
	})
}

// ── Logout ────────────────────────────────────────────────────────────────────

// LogoutHandler revokes the refresh token from the DB and clears the cookie.
// The access token expires naturally (it is short-lived by design).
func LogoutHandler(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err == nil && rawToken != "" {
		tokenHash := HashRefreshToken(rawToken)
		db := GetDB()
		if _, dbErr := db.Exec(c.Request.Context(),
			`UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1 AND revoked_at IS NULL`,
			tokenHash,
		); dbErr != nil {
			slog.Error("logout: failed to revoke refresh token", "error", dbErr)
		}
	}
	clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ── Cookie helpers ────────────────────────────────────────────────────────────

// setRefreshCookie writes the refresh token cookie using net/http directly so
// that SameSite can be set. Gin's c.SetCookie does not expose SameSite, which
// causes cross-origin cookies (e.g. Cloudflare Tunnel) to be silently dropped
// by the browser when the default SameSite=Lax policy applies.
func setRefreshCookie(c *gin.Context, rawToken string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    rawToken,
		MaxAge:   int(refreshTokenTTL.Seconds()),
		Path:     "/api/auth",
		Secure:   Config.CookieSecure,
		HttpOnly: true,
		SameSite: Config.CookieSameSite,
	})
}

func clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/auth",
		Secure:   Config.CookieSecure,
		HttpOnly: true,
		SameSite: Config.CookieSameSite,
	})
}

// ── Change Password ───────────────────────────────────────────────────────────

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

func ChangePasswordHandler(c *gin.Context) {
	u := GetAuthUser(c)

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 8 characters"})
		return
	}

	ctx := c.Request.Context()
	db := GetDB()

	var passwordHash string
	err := db.QueryRow(ctx, `
		SELECT password_hash FROM users WHERE id = $1::uuid AND active = true
	`, u.UserID).Scan(&passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		slog.Error("change-password: db error", "user_id", u.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !CheckPassword(req.CurrentPassword, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	newHash, err := HashPassword(req.NewPassword)
	if err != nil {
		slog.Error("change-password: hash failed", "user_id", u.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if _, err := db.Exec(ctx, `
		UPDATE users SET password_hash = $1, must_change_password = false WHERE id = $2::uuid
	`, newHash, u.UserID); err != nil {
		slog.Error("change-password: update failed", "user_id", u.UserID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Password changed successfully",
		"mustChangePassword": false,
	})
}

// ── Me ────────────────────────────────────────────────────────────────────────

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
		slog.Error("me: db error", "user_id", authUser.UserID, "error", err)
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

	var employeeResp interface{}
	if employeeID != nil && *employeeID != "" {
		var empID, empFirstName, empLastName, empRole string
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

// ── Register (disabled) ───────────────────────────────────────────────────────

func RegisterHandler(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Public registration is disabled",
		"message": "Contact your administrator to create an account",
	})
}

// ── Protected test routes ─────────────────────────────────────────────────────

func ProtectedMeHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{
		"user_id": u.UserID, "company_id": u.CompanyID,
		"employee_id": u.EmployeeID, "role": u.Role, "email": u.Email,
	})
}

func ProtectedDirectorEngineerHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{"message": "Access granted for direktor/inzenjer", "role": u.Role})
}

func ProtectedAdminHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{"message": "Access granted for direktor/inzenjer/administracija", "role": u.Role})
}

func ProtectedPoslovodaHandler(c *gin.Context) {
	u := GetAuthUser(c)
	c.JSON(http.StatusOK, gin.H{"message": "Access granted for direktor/inzenjer/poslovoda", "role": u.Role})
}
