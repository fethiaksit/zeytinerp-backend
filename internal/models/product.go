package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	Name          string          `json:"name" gorm:"not null"`
	Barcode       *string         `json:"barcode" gorm:"uniqueIndex"`
	Category      string          `json:"category"`
	PurchasePrice decimal.Decimal `json:"purchase_price" gorm:"type:numeric(12,2);not null"`
	SalePrice     decimal.Decimal `json:"sale_price" gorm:"type:numeric(12,2);not null"`
	CriticalStock decimal.Decimal `json:"critical_stock" gorm:"type:numeric(12,2);not null;default:0"`
	IsActive      bool            `json:"is_active" gorm:"not null;default:true"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
