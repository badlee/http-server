package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GenerateToken generates a signed JWT access token for the user.
// It generates a unique JTI and stores it in the configured database to track active sessions.
func (m *Manager) GenerateToken(user *User, expirationStr string, issuer string) (string, error) {
	if m.db == nil {
		return "", errors.New("auth database not initialized")
	}

	ttl, err := time.ParseDuration(expirationStr)
	if err != nil {
		ttl = 1 * time.Hour // Default
	}
	expiresAt := time.Now().Add(ttl)

	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	jti := strings.ReplaceAll(uid.String(), "-", "")

	// Store token in DB
	tokenRecord := AuthToken{
		JTI:       jti,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}
	if err := m.db.Create(&tokenRecord).Error; err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"jti":      jti,
		"sub":      user.ID,
		"username": user.Username,
		"email":    user.Email,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}
	if issuer != "" {
		claims["iss"] = issuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

// ValidateToken parses a JWT, verifies its signature, checks expiration, and validates
// the JTI against the database to ensure it hasn't been revoked or expired.
func (m *Manager) ValidateToken(tokenStr string) (*User, error) {
	if m.db == nil {
		return nil, errors.New("auth database not initialized")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return nil, errors.New("missing jti claim")
	}

	// Verify against database
	var tokenRecord AuthToken
	if err := m.db.Where("jti = ?", jti).First(&tokenRecord).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("token revoked or not found")
		}
		return nil, err
	}

	// Double check expiration internally as well
	if time.Now().After(tokenRecord.ExpiresAt) {
		m.db.Delete(&tokenRecord) // Clean up
		return nil, errors.New("token expired in db")
	}

	// Extract user details
	userID, _ := claims["sub"].(string)
	username, _ := claims["username"].(string)
	email, _ := claims["email"].(string)

	return &User{
		ID:       userID,
		Username: username,
		Email:    email,
	}, nil
}

// RevokeToken removes a token from the active tokens database, effectively invalidating it.
func (m *Manager) RevokeToken(jti string) error {
	if m.db == nil {
		return errors.New("auth database not initialized")
	}
	return m.db.Where("jti = ?", jti).Delete(&AuthToken{}).Error
}
