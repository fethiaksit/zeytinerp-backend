package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CustomerTransaction struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	CustomerID      uint            `json:"customer_id" gorm:"not null;index"`
	TransactionDate time.Time       `json:"transaction_date" gorm:"type:date;not null"`
	Type            string          `json:"type" gorm:"not null"`
	Amount          decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	Customer        Customer        `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
