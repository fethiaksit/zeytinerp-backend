package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type SupplierTransaction struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	SupplierID      uint            `json:"supplier_id" gorm:"not null;index"`
	TransactionDate time.Time       `json:"transaction_date" gorm:"type:date;not null"`
	Type            string          `json:"type" gorm:"not null"`
	Amount          decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	PaymentMethod   string          `json:"payment_method"`
	Note            string          `json:"note"`
	InvoiceNo       string          `json:"invoice_no"`
	CreatedAt       time.Time       `json:"created_at"`
	Supplier        Supplier        `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
