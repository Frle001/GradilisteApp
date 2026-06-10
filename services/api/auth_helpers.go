package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims contains all claims embedded in the access token.
// Always derive role, company, and user identity from these — never trust the frontend.
type JWTClaims struct {
	UserID     string `json:"user_id"`
	CompanyID  string `json:"company_id"`
	EmployeeID string `json:"employee_id,omitempty"` // empty string when user has no employee record
	Role       string `json:"role"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), Config.BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateAccessToken(userID, companyID, employeeID, role, email string) (string, error) {
	expiry, err := time.ParseDuration(Config.JWTExpiresIn)
	if err != nil {
		expiry = 15 * time.Minute
	}

	claims := JWTClaims{
		UserID:     userID,
		CompanyID:  companyID,
		EmployeeID: employeeID,
		Role:       role,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(Config.JWTSecret))
}

func ValidateAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(Config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// GenerateRefreshToken returns a random 32-byte hex token and its SHA-256 hash.
// Store the hash in the DB; send the raw token to the client via HTTP-only cookie.
func GenerateRefreshToken() (rawToken, tokenHash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(buf)
	tokenHash = HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashRefreshToken returns the hex-encoded SHA-256 of a raw refresh token.
// SHA-256 is used instead of bcrypt because refresh token lookup requires an
// exact match, not password comparison.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
