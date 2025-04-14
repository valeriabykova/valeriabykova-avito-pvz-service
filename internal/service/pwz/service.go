package pwz_service

import (
	"avito/internal/oapi"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCity        = errors.New("invalid city")
	ErrInvalidProductType = errors.New("invalid product type")

	defaultLimit     = 1000
	defaultStartDate = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	defaultEndDate   = time.Date(9999, 9, 9, 9, 9, 9, 9, time.UTC)
)

type Repository interface {
	ListPVZInfo(ctx context.Context, startDate, endDate time.Time, limit int, page int) ([]oapi.GetPvzResponseItem, error)
	CreatePVZ(ctx context.Context, city oapi.PVZCity) (oapi.PVZ, error)
	StartReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error)
	AddProduct(ctx context.Context, pvdID uuid.UUID, productType oapi.ProductType) (oapi.Product, error)
	FinishReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error)
	CheckPVZ(ctx context.Context, pvzID uuid.UUID) error
	DeleteLastProduct(ctx context.Context, pvzID uuid.UUID) error
}

type service struct {
	repo Repository
}

func New(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) CreatePvz(ctx context.Context, city oapi.PVZCity) (oapi.PVZ, error) {
	if !isValidCity(city) {
		return oapi.PVZ{}, fmt.Errorf("%w: %s", ErrInvalidCity, city)
	}
	return s.repo.CreatePVZ(ctx, city)
}

func (s *service) AddProduct(ctx context.Context, pvzID uuid.UUID, productType oapi.ProductType) (oapi.Product, error) {
	if !isValidProductType(productType) {
		return oapi.Product{}, ErrInvalidProductType
	}
	err := s.repo.CheckPVZ(ctx, pvzID)
	if err != nil {
		return oapi.Product{}, err
	}

	return s.repo.AddProduct(ctx, pvzID, productType)
}

func (s *service) StartReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error) {
	err := s.repo.CheckPVZ(ctx, pvzID)
	if err != nil {
		return oapi.Reception{}, err
	}

	return s.repo.StartReception(ctx, pvzID)
}

func (s *service) FinishReception(ctx context.Context, pvzID uuid.UUID) (oapi.Reception, error) {
	err := s.repo.CheckPVZ(ctx, pvzID)
	if err != nil {
		return oapi.Reception{}, err
	}

	return s.repo.FinishReception(ctx, pvzID)
}

func (s *service) DeleteLastProduct(ctx context.Context, pvzID uuid.UUID) error {
	err := s.repo.CheckPVZ(ctx, pvzID)
	if err != nil {
		return err
	}
	return s.repo.DeleteLastProduct(ctx, pvzID)
}

func (s *service) ListPVZInfo(ctx context.Context, params oapi.GetPvzParams) ([]oapi.GetPvzResponseItem, error) {
	if params.Page == nil {
		x := 1
		params.Page = &x
	}
	if params.Limit == nil {
		params.Limit = &defaultLimit
	}
	if params.StartDate == nil {
		params.StartDate = &defaultStartDate
	}
	if params.EndDate == nil {
		params.EndDate = &defaultEndDate
	}
	return s.repo.ListPVZInfo(ctx, *params.StartDate, *params.EndDate, *params.Limit, *params.Page)
}

func isValidCity(city oapi.PVZCity) bool {
	return city == oapi.Москва || city == oapi.Казань || city == oapi.СанктПетербург
}

func isValidProductType(t oapi.ProductType) bool {
	return t == oapi.ProductTypeОбувь || t == oapi.ProductTypeОдежда || t == oapi.ProductTypeЭлектроника
}
