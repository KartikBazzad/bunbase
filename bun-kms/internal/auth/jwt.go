package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role is the token role (admin = full access, operator = encrypt/decrypt + secrets, reader = get only).
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleReader   Role = "reader"
)

// Claims holds JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role,omitempty"`
}

// ValidateToken parses and validates a JWT token string with the given secret.
func ValidateToken(tokenString string, secret []byte) (*Claims, error) {
	if len(secret) == 0 {
		return nil, errors.New("no secret configured")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// NewToken creates a new JWT token for the given subject and role.
func NewToken(secret []byte, sub, role string, expiry time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("no secret configured")
	}
	now := time.Now().UTC()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Role: role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// HasRole returns true if the claims have at least the required role (admin > operator > reader).
func (c *Claims) HasRole(required Role) bool {
	switch required {
	case RoleAdmin:
		return c.Role == string(RoleAdmin)
	case RoleOperator:
		return c.Role == string(RoleAdmin) || c.Role == string(RoleOperator)
	case RoleReader:
		return c.Role == string(RoleAdmin) || c.Role == string(RoleOperator) || c.Role == string(RoleReader)
	default:
		return false
	}
}
