package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type BankAccount struct {
	ID             uint              `json:"id" gorm:"primaryKey"`
	AccountName    string            `json:"account_name" gorm:"not null"`
	BankName       string            `json:"bank_name" gorm:"not null"`
	IBAN           string            `json:"iban"`
	OpeningBalance decimal.Decimal   `json:"opening_balance" gorm:"type:numeric(12,2);not null"`
	CurrentBalance decimal.Decimal   `json:"current_balance" gorm:"type:numeric(12,2);not null"`
	IsActive       bool              `json:"is_active" gorm:"not null;default:true"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Transactions   []BankTransaction `json:"transactions,omitempty" gorm:"foreignKey:BankAccountID;constraint:OnDelete:CASCADE"`
}

type BankTransaction struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	BankAccountID   uint            `json:"bank_account_id" gorm:"not null;index"`
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
	BankAccount     BankAccount     `json:"-" gorm:"constraint:OnDelete:CASCADE"`
}
