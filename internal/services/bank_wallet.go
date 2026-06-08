package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"market-erp-backend/internal/models"
)

type BankWalletAccountSummary struct {
	ID             uint            `json:"id"`
	AccountName    string          `json:"account_name"`
	BankName       string          `json:"bank_name"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
}

type BankWalletSummary struct {
	TotalBalance decimal.Decimal            `json:"total_balance"`
	Accounts     []BankWalletAccountSummary `json:"accounts"`
	TodayIncome  decimal.Decimal            `json:"today_income"`
	TodayOutcome decimal.Decimal            `json:"today_outcome"`
	TodayNet     decimal.Decimal            `json:"today_net"`
}

type BankPeriodSummary struct {
	Date         string                   `json:"date,omitempty"`
	Month        string                   `json:"month,omitempty"`
	Income       decimal.Decimal          `json:"income"`
	Outcome      decimal.Decimal          `json:"outcome"`
	Net          decimal.Decimal          `json:"net"`
	Transactions []models.BankTransaction `json:"transactions"`
}

func BankTransactionSignedAmount(txType string, amount decimal.Decimal) decimal.Decimal {
	switch txType {
	case "opening_balance", "cash_deposit", "pos_income", "bank_income", "transfer_in":
		return amount.Abs()
	case "payment", "expense", "transfer_out":
		return amount.Abs().Neg()
	case "correction":
		return amount
	default:
		return decimal.Zero
	}
}

func CreateBankTransaction(db *gorm.DB, transaction models.BankTransaction) (models.BankTransaction, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		var account models.BankAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, transaction.BankAccountID).Error; err != nil {
			return err
		}

		signedAmount := BankTransactionSignedAmount(transaction.TransactionType, transaction.Amount)
		transaction.BalanceAfter = account.CurrentBalance.Add(signedAmount)
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		return tx.Model(&models.BankAccount{}).
			Where("id = ?", account.ID).
			Updates(map[string]interface{}{"current_balance": transaction.BalanceAfter}).Error
	})
	return transaction, err
}

func RecalculateBankAccountBalance(db *gorm.DB, accountID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var account models.BankAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, accountID).Error; err != nil {
			return err
		}

		var transactions []models.BankTransaction
		if err := tx.Where("bank_account_id = ?", accountID).
			Order("transaction_date asc, id asc").
			Find(&transactions).Error; err != nil {
			return err
		}

		balance := decimal.Zero
		for _, transaction := range transactions {
			balance = balance.Add(BankTransactionSignedAmount(transaction.TransactionType, transaction.Amount))
			if err := tx.Model(&models.BankTransaction{}).
				Where("id = ?", transaction.ID).
				Update("balance_after", balance).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.BankAccount{}).
			Where("id = ?", accountID).
			Update("current_balance", balance).Error
	})
}

func TotalBankBalance(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(current_balance), 0)::text
		FROM bank_accounts
		WHERE is_active = true
	`)
}

func BankWalletOverview(db *gorm.DB, today time.Time) (BankWalletSummary, error) {
	var accounts []BankWalletAccountSummary
	if err := db.Model(&models.BankAccount{}).
		Select("id, account_name, bank_name, current_balance").
		Where("is_active = true").
		Order("id asc").
		Find(&accounts).Error; err != nil {
		return BankWalletSummary{}, err
	}

	total, err := TotalBankBalance(db)
	if err != nil {
		return BankWalletSummary{}, err
	}
	income, outcome, err := BankPeriodIncomeOutcome(db, today, today.AddDate(0, 0, 1))
	if err != nil {
		return BankWalletSummary{}, err
	}
	return BankWalletSummary{
		TotalBalance: total,
		Accounts:     accounts,
		TodayIncome:  income,
		TodayOutcome: outcome,
		TodayNet:     income.Sub(outcome),
	}, nil
}

func BankPeriodSummaryData(db *gorm.DB, start, end time.Time) (decimal.Decimal, decimal.Decimal, []models.BankTransaction, error) {
	income, outcome, err := BankPeriodIncomeOutcome(db, start, end)
	if err != nil {
		return decimal.Zero, decimal.Zero, nil, err
	}
	var transactions []models.BankTransaction
	if err := db.Joins("JOIN bank_accounts ba ON ba.id = bank_transactions.bank_account_id").
		Where("ba.is_active = true").
		Where("bank_transactions.transaction_date >= ? AND bank_transactions.transaction_date < ?", start, end).
		Order("bank_transactions.transaction_date desc, bank_transactions.id desc").
		Find(&transactions).Error; err != nil {
		return decimal.Zero, decimal.Zero, nil, err
	}
	return income, outcome, transactions, nil
}

func BankPeriodIncomeOutcome(db *gorm.DB, start, end time.Time) (decimal.Decimal, decimal.Decimal, error) {
	var rows []struct {
		TransactionType string
		Amount          string
	}
	if err := db.Table("bank_transactions").
		Select("bank_transactions.transaction_type, bank_transactions.amount::text AS amount").
		Joins("JOIN bank_accounts ba ON ba.id = bank_transactions.bank_account_id").
		Where("ba.is_active = true").
		Where("bank_transactions.transaction_type <> ?", "opening_balance").
		Where("bank_transactions.transaction_date >= ? AND bank_transactions.transaction_date < ?", start, end).
		Find(&rows).Error; err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	income := decimal.Zero
	outcome := decimal.Zero
	for _, row := range rows {
		amount, err := decimal.NewFromString(row.Amount)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		signed := BankTransactionSignedAmount(row.TransactionType, amount)
		if signed.IsPositive() {
			income = income.Add(signed)
		}
		if signed.IsNegative() {
			outcome = outcome.Add(signed.Abs())
		}
	}
	return income, outcome, nil
}
