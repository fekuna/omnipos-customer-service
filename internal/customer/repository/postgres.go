package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fekuna/omnipos-customer-service/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, customer *model.Customer) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Customer, error)
	GetByPhone(ctx context.Context, merchantID uuid.UUID, phone string) (*model.Customer, error)
	List(ctx context.Context, merchantID uuid.UUID, page, pageSize int, search string) ([]*model.Customer, int, error)
	Update(ctx context.Context, customer *model.Customer) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddLoyaltyPoints(ctx context.Context, id uuid.UUID, points int32) error
}

type pgRepository struct {
	db *sqlx.DB
}

func NewPGRepository(db *sqlx.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, c *model.Customer) error {
	query := `
		INSERT INTO customers (id, merchant_id, name, phone, email, address, loyalty_points, created_at, updated_at)
		VALUES (:id, :merchant_id, :name, :phone, :email, :address, :loyalty_points, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, c)
	return err
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Customer, error) {
	var c model.Customer
	query := `SELECT * FROM customers WHERE id = $1`
	if err := r.db.GetContext(ctx, &c, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *pgRepository) GetByPhone(ctx context.Context, merchantID uuid.UUID, phone string) (*model.Customer, error) {
	var c model.Customer
	query := `SELECT * FROM customers WHERE merchant_id = $1 AND phone = $2`
	if err := r.db.GetContext(ctx, &c, query, merchantID, phone); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *pgRepository) List(ctx context.Context, merchantID uuid.UUID, page, pageSize int, search string) ([]*model.Customer, int, error) {
	offset := (page - 1) * pageSize
	customers := []*model.Customer{}
	var total int

	// Base query
	query := `SELECT * FROM customers WHERE merchant_id = $1`
	countQuery := `SELECT COUNT(*) FROM customers WHERE merchant_id = $1`
	args := []interface{}{merchantID}

	// Add search if present
	if search != "" {
		wildcard := "%" + search + "%"
		// Append params safely.
		// Params: $1 is merchantID. Next are $2 and $3.
		query += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d)", len(args)+1, len(args)+2)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d)", len(args)+1, len(args)+2)
		args = append(args, wildcard, wildcard)
	}

	// Order and Limit
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	argsList := append(args, pageSize, offset)

	if err := r.db.SelectContext(ctx, &customers, query, argsList...); err != nil {
		return nil, 0, err
	}

	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

func (r *pgRepository) Update(ctx context.Context, c *model.Customer) error {
	c.UpdatedAt = time.Now()
	query := `
		UPDATE customers 
		SET name = :name, phone = :phone, email = :email, address = :address, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, c)
	return err
}

func (r *pgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM customers WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *pgRepository) AddLoyaltyPoints(ctx context.Context, id uuid.UUID, points int32) error {
	query := `
		UPDATE customers 
		SET loyalty_points = loyalty_points + $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, points, time.Now(), id)
	return err
}
