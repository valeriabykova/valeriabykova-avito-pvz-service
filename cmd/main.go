package main

import (
	"avito/internal/oapi"
	auth_repository "avito/internal/repository/auth"
	pwz_repository "avito/internal/repository/pwz"
	"avito/internal/server"
	auth_service "avito/internal/service/auth"
	"avito/internal/service/auth/jwt"
	pwz_service "avito/internal/service/pwz"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	authRepo := auth_repository.New(pool)
	jwtHelper := jwt.New("aboba")
	authService := auth_service.New(authRepo, jwtHelper)

	pwzRepo := pwz_repository.New(pool)
	pwzService := pwz_service.New(pwzRepo)

	server := server.New(authService, pwzService)

	r := gin.Default()
	config := cors.Config{
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowAllOrigins: true,
		MaxAge:          12 * time.Hour,
	}
	r.Use(cors.New(config))
	oapi.RegisterHandlers(r, server)
	r.Run()
}
