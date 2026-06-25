package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type WalletTransaction struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	TransactionDate time.Time       `json:"transaction_date" gorm:"type:date;not null"`
	TransactionType string          `json:"transaction_type" gorm:"not null"`
	Amount          decimal.Decimal `json:"amount" gorm:"type:numeric(12,2);not null"`
	BalanceAfter    decimal.Decimal `json:"balance_after" gorm:"type:numeric(12,2);not null"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	RelatedType     string          `json:"related_type"`
	RelatedID       *uint           `json:"related_id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}
