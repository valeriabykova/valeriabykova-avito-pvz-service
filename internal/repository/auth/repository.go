package auth_repository

import (
	"avito/internal/oapi"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserDoesNotExist  = errors.New("user does no exist")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) GetUserRole(ctx context.Context, email string, password string) (oapi.UserRole, error) {
	var role oapi.UserRole
	var passwordHash string

	err := r.pool.QueryRow(ctx, `
		SELECT password_hash, role FROM users 
		WHERE email = $1
	`, email).Scan(&passwordHash, &role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrUserDoesNotExist
		}
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return "", ErrUserDoesNotExist
	}

	return role, nil
}

func (r *repository) CreateUser(ctx context.Context, role oapi.UserRole, email string, passwordHash string) (types.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING
		returning id
	`, email, passwordHash, role).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return id, ErrUserAlreadyExists
		}
		return id, err
	}
	return id, nil
}

func checkPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
