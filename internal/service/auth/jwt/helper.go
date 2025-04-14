package jwt

import (
	"avito/internal/oapi"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type helper struct {
	secret []byte
}

func New(secretKey string) *helper {
	return &helper{secret: []byte(secretKey)}
}

// Function to create JWT tokens with claims
func (h *helper) NewToken(role oapi.UserRole) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})

	signed, err := token.SignedString(h.secret)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}
	return signed, nil
}

func (h *helper) verifyToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return h.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token '%s': %w", tokenString, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return token, nil
}

func (h *helper) GetRoleClaim(tokenString string) (oapi.UserRole, error) {
	token, err := h.verifyToken(tokenString)
	if err != nil {
		println("error verifying token: ", err.Error())
		return "", fmt.Errorf("%w: error verifying token", ErrInvalidToken)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("%w: claims have to be jwt.MapClaims", ErrInvalidToken)
	}

	role, ok := claims["role"]
	if !ok {
		return "", fmt.Errorf("%w: field `role` not in claims", ErrInvalidToken)
	}

	roleString, ok := role.(string)
	if !ok {
		return "", fmt.Errorf("%w: field `role` is not string", ErrInvalidToken)
	}

	return oapi.UserRole(roleString), nil
}
