package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Customer struct {
	ID           uint             `json:"id" gorm:"primaryKey"`
	Name         string           `json:"name" gorm:"not null"`
	Phone        string           `json:"phone"`
	Address      string           `json:"address"`
	CustomerType string           `json:"customer_type" gorm:"not null;default:cari"`
	Note         string           `json:"note"`
	IsActive     bool             `json:"is_active" gorm:"not null"`
	CreditLimit  *decimal.Decimal `json:"credit_limit"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}
