package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type FinancialDebt struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	DebtType        string          `json:"debt_type" gorm:"not null"`
	InstitutionName string          `json:"institution_name" gorm:"not null"`
	Title           string          `json:"title" gorm:"not null"`
	TotalAmount     decimal.Decimal `json:"total_amount" gorm:"type:numeric(12,2);not null"`
	RemainingAmount decimal.Decimal `json:"remaining_amount" gorm:"-"`
	StartDate       time.Time       `json:"start_date" gorm:"type:date;not null"`
	EndDate         time.Time       `json:"end_date" gorm:"type:date;not null"`
	DueDate         time.Time       `json:"-" gorm:"column:due_date;type:date;not null"`
	Status          string          `json:"status" gorm:"not null;default:active"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type FinancialDebtInstallment struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	FinancialDebtID uint            `json:"financial_debt_id" gorm:"not null;index"`
	InstallmentNo   int             `json:"installment_no" gorm:"not null"`
	DueDate         time.Time       `json:"due_date" gorm:"type:date;not null"`
	Amount          decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	PaidAmount      decimal.Decimal `json:"paid_amount" gorm:"type:numeric(12,2);not null;default:0"`
	Status          string          `json:"status" gorm:"not null;default:pending"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	FinancialDebt   FinancialDebt   `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}

type FinancialDebtPayment struct {
	ID              uint                     `json:"id" gorm:"primaryKey"`
	FinancialDebtID uint                     `json:"financial_debt_id" gorm:"not null;index"`
	InstallmentID   *uint                    `json:"installment_id,omitempty" gorm:"index"`
	PaymentDate     time.Time                `json:"payment_date" gorm:"type:date;not null"`
	Amount          decimal.Decimal          `json:"amount" gorm:"type:numeric(12,2);not null"`
	PaymentMethod   string                   `json:"payment_method" gorm:"not null"`
	Note            string                   `json:"note"`
	CreatedAt       time.Time                `json:"created_at"`
	FinancialDebt   FinancialDebt            `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Installment     FinancialDebtInstallment `json:"-" gorm:"foreignKey:InstallmentID;constraint:OnDelete:CASCADE"`
}
