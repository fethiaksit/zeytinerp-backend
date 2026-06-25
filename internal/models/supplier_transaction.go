package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type SupplierTransaction struct {
	ID              uint                      `json:"id" gorm:"primaryKey"`
	SupplierID      uint                      `json:"supplier_id" gorm:"not null;index"`
	TransactionDate time.Time                 `json:"transaction_date" gorm:"type:date;not null"`
	Type            string                    `json:"type" gorm:"not null"`
	Amount          decimal.Decimal           `json:"amount" gorm:"type:numeric(12,2);not null"`
	Currency        string                    `json:"currency" gorm:"type:text;not null;default:TRY"`
	ExchangeRate    decimal.Decimal           `json:"exchange_rate" gorm:"type:numeric(18,6);not null;default:1"`
	AmountOriginal  decimal.Decimal           `json:"amount_original" gorm:"type:numeric(14,2);not null"`
	AmountTRY       decimal.Decimal           `json:"amount_try" gorm:"type:numeric(14,2);not null"`
	PaymentMethod   string                    `json:"payment_method"`
	Note            string                    `json:"note"`
	InvoiceNo       string                    `json:"invoice_no"`
	ImageURL        string                    `json:"image_url"`
	FilePath        string                    `json:"file_path"`
	CreatedAt       time.Time                 `json:"created_at"`
	Supplier        Supplier                  `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Files           []SupplierTransactionFile `json:"files,omitempty" gorm:"foreignKey:SupplierTransactionID;constraint:OnDelete:CASCADE"`
}

type ExchangeRate struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	CurrencyCode string          `json:"currency_code" gorm:"not null;index"`
	RateToTRY    decimal.Decimal `json:"rate_to_try" gorm:"type:numeric(18,6);not null"`
	Source       string          `json:"source" gorm:"not null"`
	RateDate     time.Time       `json:"rate_date" gorm:"type:date;not null"`
	CreatedAt    time.Time       `json:"created_at"`
}

type SupplierTransactionFile struct {
	ID                    uint                `json:"id" gorm:"primaryKey"`
	SupplierTransactionID uint                `json:"supplier_transaction_id" gorm:"not null;index"`
	FileName              string              `json:"file_name" gorm:"not null"`
	FilePath              string              `json:"file_path" gorm:"not null"`
	FileURL               string              `json:"file_url" gorm:"not null"`
	MimeType              string              `json:"mime_type" gorm:"not null"`
	Size                  int64               `json:"size" gorm:"not null"`
	PageOrder             int                 `json:"page_order" gorm:"not null;default:0"`
	CreatedAt             time.Time           `json:"created_at"`
	SupplierTransaction   SupplierTransaction `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
