package model

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID            uuid.UUID `db:"id"`
	MerchantID    uuid.UUID `db:"merchant_id"`
	Name          string    `db:"name"`
	Phone         string    `db:"phone"`
	Email         string    `db:"email"`
	Address       string    `db:"address"`
	LoyaltyPoints int32     `db:"loyalty_points"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
