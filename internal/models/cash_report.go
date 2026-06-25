package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CashReport struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	ReportDate       time.Time       `json:"report_date" gorm:"type:date;not null"`
	CashAmount       decimal.Decimal `json:"cash_amount" gorm:"type:numeric(12,2);not null"`
	PosAmount        decimal.Decimal `json:"pos_amount" gorm:"type:numeric(12,2);not null"`
	QRAmount         decimal.Decimal `json:"qr_amount" gorm:"column:qr_amount;type:numeric(12,2);not null"`
	CreditCollection decimal.Decimal `json:"credit_collection" gorm:"type:numeric(12,2);not null"`
	CreditGiven      decimal.Decimal `json:"credit_given" gorm:"type:numeric(12,2);not null"`
	Note             string          `json:"note"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (CashReport) TableName() string {
	return "daily_cash_reports"
}
