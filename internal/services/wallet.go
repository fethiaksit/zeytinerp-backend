package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type WalletSummary struct {
	CurrentBalance decimal.Decimal `json:"current_balance"`
	TodayIncome    decimal.Decimal `json:"today_income"`
	TodayExpense   decimal.Decimal `json:"today_expense"`
	TodayNet       decimal.Decimal `json:"today_net"`
}

func WalletTransactionSignedAmount(txType string, amount decimal.Decimal) decimal.Decimal {
	switch txType {
	case "opening_balance", "cash_income", "cash_sale", "pos_income", "bank_income", "cash_deposit":
		return amount.Abs()
	case "payment", "expense", "cash_withdraw":
		return amount.Abs().Neg()
	case "correction":
		return amount
	default:
		return decimal.Zero
	}
}

func CreateWalletTransaction(db *gorm.DB, transaction models.WalletTransaction) (models.WalletTransaction, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockWalletTransactions(tx); err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		if err := RecalculateWalletBalances(tx); err != nil {
			return err
		}
		return tx.First(&transaction, transaction.ID).Error
	})
	return transaction, err
}

func DeleteWalletTransaction(db *gorm.DB, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockWalletTransactions(tx); err != nil {
			return err
		}
		if err := tx.Delete(&models.WalletTransaction{}, id).Error; err != nil {
			return err
		}
		return RecalculateWalletBalances(tx)
	})
}

func RecalculateWalletBalances(db *gorm.DB) error {
	var transactions []models.WalletTransaction
	if err := db.Order("transaction_date asc, id asc").Find(&transactions).Error; err != nil {
		return err
	}

	balance := decimal.Zero
	for _, transaction := range transactions {
		balance = balance.Add(WalletTransactionSignedAmount(transaction.TransactionType, transaction.Amount))
		if err := db.Model(&models.WalletTransaction{}).
			Where("id = ?", transaction.ID).
			Update("balance_after", balance).Error; err != nil {
			return err
		}
	}
	return nil
}

func WalletOverview(db *gorm.DB, today time.Time) (WalletSummary, error) {
	currentBalance, err := WalletCurrentBalance(db)
	if err != nil {
		return WalletSummary{}, err
	}
	todayIncome, todayExpense, err := WalletPeriodIncomeExpense(db, today, today.AddDate(0, 0, 1))
	if err != nil {
		return WalletSummary{}, err
	}
	return WalletSummary{
		CurrentBalance: currentBalance,
		TodayIncome:    todayIncome,
		TodayExpense:   todayExpense,
		TodayNet:       todayIncome.Sub(todayExpense),
	}, nil
}

func WalletCurrentBalance(db *gorm.DB) (decimal.Decimal, error) {
	var rows []struct {
		TransactionType string
		Amount          string
	}
	if err := db.Table("wallet_transactions").
		Select("transaction_type, amount::text AS amount").
		Find(&rows).Error; err != nil {
		return decimal.Zero, err
	}
	return sumWalletRows(rows)
}

func WalletPeriodIncomeExpense(db *gorm.DB, start, end time.Time) (decimal.Decimal, decimal.Decimal, error) {
	var rows []struct {
		TransactionType string
		Amount          string
	}
	if err := db.Table("wallet_transactions").
		Select("transaction_type, amount::text AS amount").
		Where("transaction_date >= ? AND transaction_date < ?", start, end).
		Where("transaction_type <> ?", "opening_balance").
		Find(&rows).Error; err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	income := decimal.Zero
	expense := decimal.Zero
	for _, row := range rows {
		amount, err := decimal.NewFromString(row.Amount)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		signed := WalletTransactionSignedAmount(row.TransactionType, amount)
		if signed.IsPositive() {
			income = income.Add(signed)
		}
		if signed.IsNegative() {
			expense = expense.Add(signed.Abs())
		}
	}
	return income, expense, nil
}

func sumWalletRows(rows []struct {
	TransactionType string
	Amount          string
}) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, row := range rows {
		amount, err := decimal.NewFromString(row.Amount)
		if err != nil {
			return decimal.Zero, err
		}
		total = total.Add(WalletTransactionSignedAmount(row.TransactionType, amount))
	}
	return total, nil
}

func lockWalletTransactions(db *gorm.DB) error {
	return db.Exec("LOCK TABLE wallet_transactions IN EXCLUSIVE MODE").Error
}
