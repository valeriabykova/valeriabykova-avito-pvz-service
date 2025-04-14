package server

import (
	"context"
	"errors"
	"log"
	"net/http"

	"avito/internal/oapi"
	auth_repository "avito/internal/repository/auth"
	pwz_repository "avito/internal/repository/pwz"
	auth_service "avito/internal/service/auth"
	"avito/internal/service/auth/jwt"
	pwz_service "avito/internal/service/pwz"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type server struct {
	authSvc AuthService
	pwzSvc  PvzService
}

type PvzService interface {
	CreatePvz(ctx context.Context, city oapi.PVZCity) (oapi.PVZ, error)
	StartReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error)
	FinishReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error)
	AddProduct(ctx context.Context, pvzID uuid.UUID, productType oapi.ProductType) (oapi.Product, error)
	DeleteLastProduct(ctx context.Context, pvzID uuid.UUID) error
	ListPVZInfo(ctx context.Context, params oapi.GetPvzParams) ([]oapi.GetPvzResponseItem, error)
}

type AuthService interface {
	GetTokenForRole(ctx context.Context, role oapi.UserRole) (string, error)
	GetRoleFromReq(bearerToken string) (oapi.UserRole, error)
	Login(ctx context.Context, email string, password string) (string, error)
	Register(ctx context.Context, role oapi.UserRole, email string, password string) (oapi.User, error)
}

func New(authSvc AuthService, pwzSvc PvzService) oapi.ServerInterface {
	return &server{
		authSvc: authSvc,
		pwzSvc:  pwzSvc,
	}
}

func (s *server) PostDummyLogin(c *gin.Context) {
	var req oapi.PostDummyLoginJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}

	token, err := s.authSvc.GetTokenForRole(c, oapi.UserRole(req.Role))
	if err != nil {
		s.setError(c, err)
		return
	}

	c.JSON(http.StatusOK, oapi.Token(token))
}

// Авторизация пользователя
// (POST /login)
func (s *server) PostLogin(c *gin.Context) {
	var req oapi.PostLoginJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}

	token, err := s.authSvc.Login(c, string(req.Email), req.Password)
	if err != nil {
		s.setError(c, err)
		return
	}

	c.JSON(http.StatusOK, oapi.Token(token))
}

// Регистрация пользователя
// (POST /register)
func (s *server) PostRegister(c *gin.Context) {
	var req oapi.PostRegisterJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: "password has to be at least 6 symbols long"})
		return
	}
	if len(req.Email) < 5 {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: "email has to be at least 5 symbols long"})
		return
	}
	user, err := s.authSvc.Register(c, oapi.UserRole(req.Role), string(req.Email), req.Password)
	if err != nil {
		s.setError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Добавление товара в текущую приемку (только для сотрудников ПВЗ)
// (POST /products)
func (s *server) PostProducts(c *gin.Context) {
	if !s.checkRole(c, oapi.UserRoleEmployee) {
		return
	}
	var req oapi.PostProductsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}
	product, err := s.pwzSvc.AddProduct(c, req.PvzId, oapi.ProductType(req.Type))
	if err != nil {
		s.setError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Получение списка ПВЗ с фильтрацией по дате приемки и пагинацией
// (GET /pvz)
func (s *server) GetPvz(c *gin.Context, params oapi.GetPvzParams) {
	if !s.checkRoleIn(c, []oapi.UserRole{oapi.UserRoleEmployee, oapi.UserRoleModerator}) {
		return
	}

	resp, err := s.pwzSvc.ListPVZInfo(c, params)
	if err != nil {
		s.setError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Создание ПВЗ (только для модераторов)
// (POST /pvz)
func (s *server) PostPvz(c *gin.Context) {
	if !s.checkRole(c, oapi.UserRoleModerator) {
		return
	}

	var req oapi.PostPvzJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}

	pvz, err := s.pwzSvc.CreatePvz(c, req.City)
	if err != nil {
		s.setError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pvz)
}

// Закрытие последней открытой приемки товаров в рамках ПВЗ
// (POST /pvz/{pvzId}/close_last_reception)
func (s *server) PostPvzPvzIdCloseLastReception(c *gin.Context, pvzId openapi_types.UUID) {
	if !s.checkRole(c, oapi.UserRoleEmployee) {
		return
	}
	reception, err := s.pwzSvc.FinishReception(c, pvzId)
	if err != nil {
		s.setError(c, err)
		return
	}
	c.JSON(http.StatusOK, reception)
}

// Удаление последнего добавленного товара из текущей приемки (LIFO, только для сотрудников ПВЗ)
// (POST /pvz/{pvzId}/delete_last_product)
func (s *server) PostPvzPvzIdDeleteLastProduct(c *gin.Context, pvzId openapi_types.UUID) {
	if !s.checkRole(c, oapi.UserRoleEmployee) {
		return
	}
	err := s.pwzSvc.DeleteLastProduct(c, pvzId)
	if err != nil {
		s.setError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// Создание новой приемки товаров (только для сотрудников ПВЗ)
// (POST /receptions)
func (s *server) PostReceptions(c *gin.Context) {
	if !s.checkRole(c, oapi.UserRoleEmployee) {
		return
	}

	var req oapi.PostReceptionsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
		return
	}

	reception, err := s.pwzSvc.StartReception(c, req.PvzId)
	if err != nil {
		s.setError(c, err)
		return
	}
	c.JSON(http.StatusOK, reception)
}

func (s *server) setError(c *gin.Context, err error) {
	badRequestErrors := []error{
		jwt.ErrInvalidToken,
		auth_service.ErrInvalidRole,
		auth_repository.ErrUserDoesNotExist,
		auth_repository.ErrUserAlreadyExists,
		pwz_service.ErrInvalidCity,
		pwz_repository.ErrPVZDoesNotExist,
		pwz_repository.ErrReceptionAlreadyExists,
		pwz_repository.ErrReceptionDoesNotExist,
		pwz_repository.ErrProductDoesNotExist,
	}
	for _, e := range badRequestErrors {
		if errors.Is(err, e) {
			c.JSON(http.StatusBadRequest, oapi.Error{Message: err.Error()})
			return
		}
	}
	log.Printf("internal error: %s", err.Error())
	c.JSON(http.StatusInternalServerError, oapi.Error{Message: "internal server error"})
}

func (s *server) checkRoleIn(c *gin.Context, required []oapi.UserRole) bool {
	role, err := s.authSvc.GetRoleFromReq(c.GetHeader("Authorization"))
	if err != nil {
		s.setError(c, err)
		return false
	}
	for _, req := range required {
		if role == req {
			return true
		}
	}
	c.JSON(http.StatusUnauthorized, oapi.Error{Message: "unauthorized"})
	return false
}

func (s *server) checkRole(c *gin.Context, required oapi.UserRole) bool {
	return s.checkRoleIn(c, []oapi.UserRole{required})
}
