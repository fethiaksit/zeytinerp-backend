package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type EmployeeTransaction struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	EmployeeID      uint            `json:"employee_id" gorm:"not null;index"`
	TransactionDate time.Time       `json:"transaction_date" gorm:"type:date;not null"`
	Type            string          `json:"type" gorm:"not null"`
	WorkDays        decimal.Decimal `json:"work_days" gorm:"type:numeric(8,2);not null;default:0"`
	Amount          decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null;default:0"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	Employee        Employee        `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
