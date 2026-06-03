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
	PaymentMethod   string                    `json:"payment_method"`
	Note            string                    `json:"note"`
	InvoiceNo       string                    `json:"invoice_no"`
	CreatedAt       time.Time                 `json:"created_at"`
	Supplier        Supplier                  `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Files           []SupplierTransactionFile `json:"files,omitempty" gorm:"foreignKey:SupplierTransactionID;constraint:OnDelete:CASCADE"`
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
