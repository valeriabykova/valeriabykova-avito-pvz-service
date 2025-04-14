package pwz_service

import (
	"avito/internal/oapi"
	pwz_repository "avito/internal/repository/pwz"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_service_CreatePvz(t *testing.T) {
	repo := NewMockRepository(t)
	repo.On("CreatePVZ", mock.Anything, oapi.Москва).Return(oapi.PVZ{}, nil)
	service := New(repo)
	ctx := context.Background()

	_, err := service.CreatePvz(ctx, "not a city")
	assert.ErrorIs(t, err, ErrInvalidCity)

	city, err := service.CreatePvz(ctx, oapi.Москва)
	assert.NoError(t, err)
	assert.Equal(t, oapi.PVZ{}, city)
}

func Test_service_StartReception(t *testing.T) {
	repo := NewMockRepository(t)
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(pwz_repository.ErrPVZDoesNotExist).Once()
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(nil)
	repo.On("StartReception", mock.Anything, mock.Anything).Return(oapi.Reception{}, nil)
	service := New(repo)
	ctx := context.Background()

	_, err := service.StartReception(ctx, uuid.UUID{})
	assert.ErrorIs(t, err, pwz_repository.ErrPVZDoesNotExist)

	reception, err := service.StartReception(ctx, uuid.UUID{})
	assert.NoError(t, err)
	assert.Equal(t, oapi.Reception{}, reception)
}

func Test_service_AddProduct(t *testing.T) {
	repo := NewMockRepository(t)
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(pwz_repository.ErrPVZDoesNotExist).Once()
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(nil)
	repo.On("AddProduct", mock.Anything, mock.Anything, mock.Anything).Return(oapi.Product{}, nil)
	service := New(repo)
	ctx := context.Background()

	_, err := service.AddProduct(ctx, uuid.UUID{}, "aboba")
	assert.ErrorIs(t, err, ErrInvalidProductType)

	_, err = service.AddProduct(ctx, uuid.UUID{}, oapi.ProductTypeЭлектроника)
	assert.ErrorIs(t, err, pwz_repository.ErrPVZDoesNotExist)

	reception, err := service.AddProduct(ctx, uuid.UUID{}, oapi.ProductTypeЭлектроника)
	assert.NoError(t, err)
	assert.Equal(t, oapi.Product{}, reception)
}

func Test_service_FinishReception(t *testing.T) {
	repo := NewMockRepository(t)
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(pwz_repository.ErrPVZDoesNotExist).Once()
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(nil)
	repo.On("FinishReception", mock.Anything, mock.Anything).Return(oapi.Reception{}, nil)
	service := New(repo)
	ctx := context.Background()

	_, err := service.FinishReception(ctx, uuid.UUID{})
	assert.ErrorIs(t, err, pwz_repository.ErrPVZDoesNotExist)

	reception, err := service.FinishReception(ctx, uuid.UUID{})
	assert.NoError(t, err)
	assert.Equal(t, oapi.Reception{}, reception)
}

func Test_service_DeleteLastProduct(t *testing.T) {
	repo := NewMockRepository(t)
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(pwz_repository.ErrPVZDoesNotExist).Once()
	repo.On("CheckPVZ", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteLastProduct", mock.Anything, mock.Anything).Return(pwz_repository.ErrProductDoesNotExist).Once()
	repo.On("DeleteLastProduct", mock.Anything, mock.Anything).Return(nil)
	service := New(repo)
	ctx := context.Background()

	err := service.DeleteLastProduct(ctx, uuid.UUID{})
	assert.ErrorIs(t, err, pwz_repository.ErrPVZDoesNotExist)

	err = service.DeleteLastProduct(ctx, uuid.UUID{})
	assert.ErrorIs(t, err, pwz_repository.ErrProductDoesNotExist)

	err = service.DeleteLastProduct(ctx, uuid.UUID{})
	assert.NoError(t, err)
}

func Test_service_ListPVZInfo(t *testing.T) {
	repo := NewMockRepository(t)
	expected := []oapi.GetPvzResponseItem{{Pvz: oapi.PVZ{}}}
	repo.On("ListPVZInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expected, nil)
	service := New(repo)
	ctx := context.Background()

	result, err := service.ListPVZInfo(ctx, oapi.GetPvzParams{})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
