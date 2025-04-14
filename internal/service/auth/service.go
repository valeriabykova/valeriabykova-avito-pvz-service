package auth_service

import (
	oapi "avito/internal/oapi"
	"avito/internal/service/auth/jwt"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	GetUserRole(ctx context.Context, email string, password string) (oapi.UserRole, error)
	CreateUser(ctx context.Context, role oapi.UserRole, email string, password string) (types.UUID, error)
}

type JWTHelper interface {
	NewToken(role oapi.UserRole) (string, error)
	GetRoleClaim(tokenString string) (oapi.UserRole, error)
}

var (
	ErrInvalidRole = errors.New("invalid role")
)

type service struct {
	repo      Repository
	jwtHelper JWTHelper
}

func New(repo Repository, jwtHelper JWTHelper) *service {
	return &service{repo: repo, jwtHelper: jwtHelper}
}

func (s *service) GetTokenForRole(ctx context.Context, role oapi.UserRole) (string, error) {
	if !validRole(role) {
		return "", ErrInvalidRole
	}
	token, err := s.jwtHelper.NewToken(role)
	if err != nil {
		return "", fmt.Errorf("error creating token: %w", err)
	}
	return token, nil
}

func (s *service) GetRoleFromReq(bearerToken string) (oapi.UserRole, error) {
	if !strings.HasPrefix(bearerToken, "Bearer ") {
		return "", fmt.Errorf("%w: Authorization has to start with `Bearer`", jwt.ErrInvalidToken)
	}
	bearerToken = strings.TrimPrefix(bearerToken, "Bearer ")
	role, err := s.jwtHelper.GetRoleClaim(bearerToken)
	if err != nil {
		return "", fmt.Errorf("error getting role from token: %w", err)
	}
	return role, nil
}

func (s *service) Login(ctx context.Context, email string, password string) (string, error) {
	role, err := s.repo.GetUserRole(ctx, email, password)
	if err != nil {
		return "", err
	}
	return s.GetTokenForRole(ctx, role)
}

func (s *service) Register(ctx context.Context, role oapi.UserRole, email string, password string) (oapi.User, error) {
	if !validRole(role) {
		return oapi.User{}, ErrInvalidRole
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return oapi.User{}, err
	}

	id, err := s.repo.CreateUser(ctx, role, email, string(passwordHash))
	if err != nil {
		return oapi.User{}, fmt.Errorf("error creating user: %w", err)
	}

	user := oapi.User{
		Id:    &id,
		Email: types.Email(email),
		Role:  role,
	}
	return user, nil
}

func validRole(role oapi.UserRole) bool {
	return role == oapi.UserRoleEmployee || role == oapi.UserRoleModerator
}
