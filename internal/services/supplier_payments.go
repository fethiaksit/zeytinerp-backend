package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SupplierCashPaymentsBetween returns supplier payments that physically leave
// the cash register. Non-cash supplier payments still reduce supplier debt,
// but they must not affect cash-flow reports.
func SupplierCashPaymentsBetween(db *gorm.DB, start, end time.Time) (decimal.Decimal, error) {
	var total string
	err := db.Table("supplier_transactions").
		Select("CAST(COALESCE(SUM(amount_try), 0) AS TEXT)").
		Where("type = ?", "payment").
		Where("LOWER(TRIM(payment_method)) IN ?", []string{"cash", "nakit"}).
		Where("transaction_date >= ? AND transaction_date < ?", start, end).
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, err
	}
	if total == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(total)
}

// DailyCashOutflowBetween keeps ordinary expense records and supplier cash
// payments as separate sources, then adds each source exactly once.
func DailyCashOutflowBetween(db *gorm.DB, start, end time.Time) (decimal.Decimal, decimal.Decimal, error) {
	var expenseTotal string
	err := db.Table("expenses").
		Select("CAST(COALESCE(SUM(amount), 0) AS TEXT)").
		Where("expense_date >= ? AND expense_date < ?", start, end).
		Scan(&expenseTotal).Error
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	expenses := decimal.Zero
	if expenseTotal != "" {
		expenses, err = decimal.NewFromString(expenseTotal)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
	}

	supplierCashPayments, err := SupplierCashPaymentsBetween(db, start, end)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return expenses.Add(supplierCashPayments), supplierCashPayments, nil
}
