package integration

import (
	"avito/internal/oapi"
	pwz_repository "avito/internal/repository/pwz"
	pwz_service "avito/internal/service/pwz"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Первым делом создает новый ПВЗ
// Добавляет новую приёмку заказов
// Добавляет 50 товаров в рамках текущей приёмки заказов
// Закрывает приёмку заказов

func TestServer(t *testing.T) {
	ctx := context.Background()
	pool := PreparePostgres(ctx, t)

	repo := pwz_repository.New(pool)
	service := pwz_service.New(repo)

	pvz, err := service.CreatePvz(ctx, oapi.Москва)
	assert.NoError(t, err)

	reception, err := service.StartReception(ctx, *pvz.Id)
	assert.NoError(t, err)

	for i := 0; i < 50; i++ {
		_, err := service.AddProduct(ctx, *pvz.Id, oapi.ProductTypeЭлектроника)
		assert.NoError(t, err)
	}

	closed, err := service.FinishReception(ctx, *pvz.Id)
	assert.NoError(t, err)
	assert.Equal(t, reception.Id, closed.Id)

	data, err := service.ListPVZInfo(ctx, oapi.GetPvzParams{})
	assert.Equal(t, len(data[0].Receptions[0].Products), 50)
}
