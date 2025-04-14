package pwz_repository

import (
	"avito/internal/oapi"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrReceptionAlreadyExists = errors.New("active reception already exists")
	ErrPVZDoesNotExist        = errors.New("pvz does not exist")
	ErrReceptionDoesNotExist  = errors.New("there is no active reception")
	ErrProductDoesNotExist    = errors.New("product does not exist")
)

type repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) listPVZ(ctx context.Context, page int, limit int) ([]oapi.PVZ, error) {
	offset := (page - 1) * limit

	rows, err := r.pool.Query(ctx, `
		SELECT id, city, registration_date 
		FROM pvz
		ORDER BY registration_date DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query pvz: %w", err)
	}
	defer rows.Close()

	pvzs := make([]oapi.PVZ, 0, limit)
	for rows.Next() {
		pvz := oapi.PVZ{}
		if err := rows.Scan(&pvz.Id, &pvz.City, &pvz.RegistrationDate); err != nil {
			return nil, fmt.Errorf("failed to scan pvz row: %w", err)
		}
		pvzs = append(pvzs, pvz)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return pvzs, nil
}

func (r *repository) getReceptionsByPVZ(ctx context.Context, startDate, endDate time.Time, pvzIDs []uuid.UUID) (map[uuid.UUID][]oapi.Reception, error) {
	result := make(map[uuid.UUID][]oapi.Reception, len(pvzIDs))

	rows, err := r.pool.Query(ctx, `
			SELECT id, pvz_id, status, datetime 
			FROM reception
			WHERE pvz_id = ANY($1) 
			AND datetime BETWEEN $2 AND $3
			ORDER BY datetime DESC
	`, pvzIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query receptions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rec := oapi.Reception{}
		if err := rows.Scan(&rec.Id, &rec.PvzId, &rec.Status, &rec.DateTime); err != nil {
			return nil, fmt.Errorf("failed to scan reception: %w", err)
		}
		result[rec.PvzId] = append(result[rec.PvzId], rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

func (r *repository) getProductsByReceptions(ctx context.Context, receptionIDs []uuid.UUID) (map[uuid.UUID][]oapi.Product, error) {
	result := make(map[uuid.UUID][]oapi.Product)
	rows, err := r.pool.Query(ctx, `
			SELECT id, reception_id, type, datetime
			FROM product
			WHERE reception_id = ANY($1)
			ORDER BY datetime DESC
	`, receptionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		prod := oapi.Product{}
		if err := rows.Scan(&prod.Id, &prod.ReceptionId, &prod.Type, &prod.DateTime); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		result[prod.ReceptionId] = append(result[prod.ReceptionId], prod)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return result, nil
}

func (r *repository) ListPVZInfo(ctx context.Context, startDate, endDate time.Time, limit int, page int) (res []oapi.GetPvzResponseItem, err error) {
	pvzs, err := r.listPVZ(ctx, page, limit)
	if err != nil {
		return
	}

	pvzIDs := make([]uuid.UUID, len(pvzs))
	for i, pvz := range pvzs {
		pvzIDs[i] = *pvz.Id
	}

	receptionsByPVZ, err := r.getReceptionsByPVZ(ctx, startDate, endDate, pvzIDs)
	if err != nil {
		return
	}

	receptionIDs := make([]uuid.UUID, len(receptionsByPVZ))
	for _, receptions := range receptionsByPVZ {
		for _, reception := range receptions {
			receptionIDs = append(receptionIDs, *reception.Id)
		}
	}

	products, err := r.getProductsByReceptions(ctx, receptionIDs)
	if err != nil {
		return
	}

	resp := make([]oapi.GetPvzResponseItem, len(pvzs))
	for i, pvz := range pvzs {
		resp[i].Pvz = pvz

		pvzReceptions := make([]oapi.GetPvzReceptionItem, len(receptionsByPVZ[*pvz.Id]))
		for i, reception := range receptionsByPVZ[*pvz.Id] {
			pvzReceptions[i].Reception = reception
			pvzReceptions[i].Products = products[*reception.Id]
		}
		resp[i].Receptions = pvzReceptions
	}

	return resp, nil
}

func (r *repository) CreatePVZ(ctx context.Context, city oapi.PVZCity) (oapi.PVZ, error) {
	now := time.Now()
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO pvz (city, registration_date)
		VALUES ($1, $2)
		returning id
	`, city, now).Scan(&id)

	if err != nil {
		return oapi.PVZ{}, err
	}

	return oapi.PVZ{
		City:             city,
		RegistrationDate: &now,
		Id:               &id,
	}, nil
}

func (r *repository) CheckPVZ(ctx context.Context, pvzID uuid.UUID) error {
	var found bool
	err := r.pool.QueryRow(ctx, `SELECT TRUE FROM pvz WHERE id = $1`, pvzID).Scan(&found)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPVZDoesNotExist
		}
		return err
	}
	return nil
}

func (r *repository) getCurrentReceptionID(ctx context.Context, pvzID uuid.UUID) (uuid.UUID, error) {
	var receptionID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM reception where pvz_id = $1 and status = $2`,
		pvzID, oapi.InProgress,
	).Scan(&receptionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return receptionID, ErrReceptionDoesNotExist
		}
	}
	return receptionID, nil
}

func (r *repository) StartReception(ctx context.Context, pvzID uuid.UUID) (rec oapi.Reception, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		err = fmt.Errorf("error starting tx: %w", err)
		return
	}
	defer func() {
		txErr := tx.Rollback(ctx)
		if txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			err = fmt.Errorf("error rolling back tx: %w", txErr)
		}
	}()

	_, err = r.getCurrentReceptionID(ctx, pvzID)
	if err == nil {
		err = ErrReceptionAlreadyExists
		return
	}

	now := time.Now()
	rec.DateTime = now
	rec.PvzId = pvzID
	rec.Status = oapi.InProgress
	err = r.pool.QueryRow(ctx, `
		INSERT INTO reception (pvz_id, status, datetime)
		VALUES ($1, $2, $3)
		RETURNING id
	`, pvzID, oapi.InProgress, now).Scan(&rec.Id)
	if err != nil {
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return
}

func (r *repository) AddProduct(ctx context.Context, pvzID uuid.UUID, productType oapi.ProductType) (prod oapi.Product, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})
	if err != nil {
		err = fmt.Errorf("error starting tx: %w", err)
		return
	}
	defer func() {
		txErr := tx.Rollback(ctx)
		if txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			err = fmt.Errorf("error rolling back tx: %w", txErr)
		}
	}()

	receptionID, err := r.getCurrentReceptionID(ctx, pvzID)
	if err != nil {
		return
	}

	now := time.Now()
	prod.DateTime = &now
	prod.ReceptionId = receptionID
	prod.Type = productType

	err = r.pool.QueryRow(ctx, `
		INSERT INTO product (reception_id, type, datetime)
		VALUES ($1, $2, $3)
		returning id
	`, receptionID, productType, now).Scan(&prod.Id)
	if err != nil {
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return
}

func (r *repository) FinishReception(ctx context.Context, pvzID uuid.UUID) (rec oapi.Reception, err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})
	if err != nil {
		err = fmt.Errorf("error starting tx: %w", err)
		return
	}
	defer func() {
		txErr := tx.Rollback(ctx)
		if txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			err = fmt.Errorf("error rolling back tx: %w", txErr)
		}
	}()

	rec.PvzId = pvzID
	rec.Status = oapi.Close
	err = r.pool.QueryRow(ctx, `
		UPDATE reception SET status = $1
		WHERE pvz_id = $2 and status = $3
		RETURNING id, datetime
	`, oapi.Close, pvzID, oapi.InProgress).Scan(&rec.Id, &rec.DateTime)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrReceptionDoesNotExist
		}
		return
	}
	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return
}

func (r *repository) DeleteLastProduct(ctx context.Context, pvzID uuid.UUID) (err error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})
	if err != nil {
		err = fmt.Errorf("error starting tx: %w", err)
		return
	}
	defer func() {
		txErr := tx.Rollback(ctx)
		if txErr != nil && !errors.Is(txErr, pgx.ErrTxClosed) {
			err = fmt.Errorf("error rolling back tx: %w", txErr)
		}
	}()

	receptionID, err := r.getCurrentReceptionID(ctx, pvzID)
	if err != nil {
		return err
	}

	var lastProductID uuid.UUID
	err = tx.QueryRow(ctx, `
			SELECT id FROM product 
			WHERE reception_id = $1 ORDER BY datetime DESC 
			LIMIT 1 FOR UPDATE
	`, receptionID).Scan(&lastProductID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrProductDoesNotExist
		}
		return err
	}

	_, err = tx.Exec(ctx, `
			DELETE FROM product 
			WHERE id = $1
	`, lastProductID)
	if err != nil {
		return
	}
	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return
}
