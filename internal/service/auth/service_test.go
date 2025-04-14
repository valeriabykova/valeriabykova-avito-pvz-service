package auth_service

import (
	"context"
	"errors"
	"testing"

	"avito/internal/oapi"
	auth_repository "avito/internal/repository/auth"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetTokenForRole(t *testing.T) {
	repoMock := NewMockRepository(t)
	jwtMock := NewMockJWTHelper(t)
	service := New(repoMock, jwtMock)

	t.Run("valid role", func(t *testing.T) {
		role := oapi.UserRoleEmployee
		expectedToken := "test-token"

		jwtMock.EXPECT().NewToken(role).Return(expectedToken, nil)

		token, err := service.GetTokenForRole(context.Background(), role)
		require.NoError(t, err)
		require.Equal(t, expectedToken, token)
	})

	t.Run("invalid role", func(t *testing.T) {
		role := oapi.UserRole("invalid")
		_, err := service.GetTokenForRole(context.Background(), role)
		require.ErrorIs(t, err, ErrInvalidRole)
	})

	t.Run("jwt helper error", func(t *testing.T) {
		role := oapi.UserRoleModerator
		expectedError := errors.New("jwt error")

		jwtMock.EXPECT().NewToken(role).Return("", expectedError)

		_, err := service.GetTokenForRole(context.Background(), role)
		require.ErrorContains(t, err, "error creating token")
	})
}

func TestService_GetRoleFromReq(t *testing.T) {
	repoMock := NewMockRepository(t)
	jwtMock := NewMockJWTHelper(t)
	service := New(repoMock, jwtMock)

	t.Run("valid token", func(t *testing.T) {
		token := "Bearer valid-token"
		expectedRole := oapi.UserRoleModerator

		jwtMock.EXPECT().
			GetRoleClaim("valid-token").
			Return(expectedRole, nil)

		role, err := service.GetRoleFromReq(token)
		require.NoError(t, err)
		require.Equal(t, expectedRole, role)
	})

	t.Run("invalid token format", func(t *testing.T) {
		token := "InvalidToken"
		_, err := service.GetRoleFromReq(token)
		require.ErrorContains(t, err, "Authorization has to start with `Bearer`")
	})

	t.Run("jwt helper error", func(t *testing.T) {
		token := "Bearer invalid-token"
		expectedError := errors.New("invalid token")

		jwtMock.EXPECT().
			GetRoleClaim("invalid-token").
			Return(oapi.UserRole(""), expectedError)

		_, err := service.GetRoleFromReq(token)
		require.ErrorContains(t, err, "error getting role from token")
	})
}

func TestService_Login(t *testing.T) {
	repoMock := NewMockRepository(t)
	jwtMock := NewMockJWTHelper(t)
	service := New(repoMock, jwtMock)

	t.Run("successful login", func(t *testing.T) {
		email := "test@example.com"
		password := "password"
		role := oapi.UserRoleEmployee
		expectedToken := "test-token"

		repoMock.EXPECT().
			GetUserRole(mock.Anything, email, password).
			Return(role, nil)

		jwtMock.EXPECT().
			NewToken(role).
			Return(expectedToken, nil)

		token, err := service.Login(context.Background(), email, password)
		require.NoError(t, err)
		require.Equal(t, expectedToken, token)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		email := "invalid@example.com"
		password := "wrong"
		expectedError := auth_repository.ErrUserDoesNotExist

		repoMock.EXPECT().
			GetUserRole(mock.Anything, email, password).
			Return(oapi.UserRole(""), expectedError)

		_, err := service.Login(context.Background(), email, password)
		require.ErrorIs(t, err, expectedError)
	})
}

func TestService_Register(t *testing.T) {
	repoMock := NewMockRepository(t)
	jwtMock := NewMockJWTHelper(t)
	service := New(repoMock, jwtMock)

	t.Run("successful registration", func(t *testing.T) {
		role := oapi.UserRoleModerator
		email := "new@example.com"
		password := "password"
		expectedID := types.UUID(uuid.New())

		repoMock.EXPECT().
			CreateUser(mock.Anything, role, email, mock.Anything).
			Return(expectedID, nil)

		user, err := service.Register(context.Background(), role, email, password)
		require.NoError(t, err)
		require.Equal(t, expectedID, *user.Id)
		require.Equal(t, email, string(user.Email))
		require.Equal(t, role, user.Role)
	})

	t.Run("invalid role", func(t *testing.T) {
		role := oapi.UserRole("invalid")
		_, err := service.Register(context.Background(), role, "test@example.com", "password")
		require.ErrorIs(t, err, ErrInvalidRole)
	})

	t.Run("duplicate email", func(t *testing.T) {
		role := oapi.UserRoleEmployee
		email := "exists@example.com"
		expectedError := auth_repository.ErrUserAlreadyExists

		repoMock.EXPECT().
			CreateUser(mock.Anything, role, email, mock.Anything).
			Return(types.UUID{}, expectedError)

		_, err := service.Register(context.Background(), role, email, "password")
		require.ErrorIs(t, err, expectedError)
	})
}
