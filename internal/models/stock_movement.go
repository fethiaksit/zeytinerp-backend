package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type StockMovement struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	ProductID    uint            `json:"product_id" gorm:"not null;index"`
	MovementDate time.Time       `json:"movement_date" gorm:"type:date;not null"`
	Type         string          `json:"type" gorm:"not null"`
	Quantity     decimal.Decimal `json:"quantity" gorm:"type:numeric(12,2);not null"`
	UnitPrice    decimal.Decimal `json:"unit_price" gorm:"type:numeric(12,2);not null"`
	Note         string          `json:"note"`
	CreatedAt    time.Time       `json:"created_at"`
	Product      Product         `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
