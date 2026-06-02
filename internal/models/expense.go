package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Expense struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	ExpenseDate   time.Time       `json:"expense_date" gorm:"type:date;not null"`
	Category      string          `json:"category" gorm:"not null"`
	Amount        decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	PaymentMethod string          `json:"payment_method"`
	Note          string          `json:"note"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
