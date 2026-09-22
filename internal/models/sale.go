package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Sale struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	SaleNo        string          `json:"sale_no" gorm:"type:varchar(80);not null;uniqueIndex"`
	PaymentMethod string          `json:"payment_method" gorm:"type:varchar(20);not null"`
	CustomerID    *uint           `json:"customer_id" gorm:"index"`
	TotalAmount   decimal.Decimal `json:"total_amount" gorm:"type:numeric(12,2);not null;default:0"`
	CreatedBy     uint            `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`

	Customer *Customer  `json:"customer,omitempty" gorm:"foreignKey:CustomerID;constraint:OnDelete:SET NULL"`
	Items    []SaleItem `json:"items" gorm:"foreignKey:SaleID;constraint:OnDelete:CASCADE"`
}

type SaleItem struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	SaleID      uint            `json:"sale_id" gorm:"not null;index"`
	ProductID   uint            `json:"product_id" gorm:"not null;index"`
	Barcode     string          `json:"barcode" gorm:"type:varchar(100);not null"`
	ProductName string          `json:"product_name" gorm:"type:varchar(255);not null"`
	Quantity    decimal.Decimal `json:"quantity" gorm:"type:numeric(12,3);not null"`
	UnitPrice   decimal.Decimal `json:"unit_price" gorm:"type:numeric(12,2);not null"`
	LineTotal   decimal.Decimal `json:"line_total" gorm:"type:numeric(12,2);not null"`
}
