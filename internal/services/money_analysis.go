package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type MoneyAnalysisBreakdownItem struct {
	Label  string          `json:"label"`
	Amount decimal.Decimal `json:"amount"`
}

type MoneyAnalysis struct {
	Month                 string                       `json:"month"`
	Income                decimal.Decimal              `json:"income"`
	Expense               decimal.Decimal              `json:"expense"`
	SupplierCashPayments  decimal.Decimal              `json:"supplier_cash_payments"`
	ExpectedBalance       decimal.Decimal              `json:"expected_balance"`
	CashBalance           decimal.Decimal              `json:"cash_balance"`
	BankBalance           decimal.Decimal              `json:"bank_balance"`
	PendingPOS            decimal.Decimal              `json:"pending_pos"`
	CustomerReceivables   decimal.Decimal              `json:"customer_receivables"`
	SupplierReceivables   decimal.Decimal              `json:"supplier_receivables"`
	EmployeeAdvances      decimal.Decimal              `json:"employee_advances"`
	AccountedTotal        decimal.Decimal              `json:"accounted_total"`
	UnexplainedDifference decimal.Decimal              `json:"unexplained_difference"`
	Breakdown             []MoneyAnalysisBreakdownItem `json:"breakdown"`
}

// MoneyAnalysisForMonth compares monthly source transactions with the current
// cash and bank positions. Wallet and bank transfers are deliberately not
// counted as income or expense because they only move the same money between
// locations and would otherwise be counted twice.
func MoneyAnalysisForMonth(db *gorm.DB, monthStart time.Time) (MoneyAnalysis, error) {
	monthEnd := monthStart.AddDate(0, 1, 0)

	income, err := moneyAnalysisIncome(db, monthStart, monthEnd)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	expense, supplierCashPayments, err := moneyAnalysisExpense(db, monthStart, monthEnd)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	cashBalance, err := moneyAnalysisWalletBalance(db)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	bankBalance, err := moneyAnalysisBankBalance(db)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	customerReceivables, err := moneyAnalysisCustomerReceivables(db)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	supplierReceivables, err := moneyAnalysisSupplierReceivables(db)
	if err != nil {
		return MoneyAnalysis{}, err
	}
	employeeAdvances, err := moneyAnalysisEmployeeAdvances(db, monthStart, monthEnd)
	if err != nil {
		return MoneyAnalysis{}, err
	}

	pendingPOS := decimal.Zero
	expectedBalance := income.Sub(expense)
	accountedTotal := cashBalance.Add(bankBalance).
		Add(pendingPOS).
		Add(customerReceivables).
		Add(supplierReceivables).
		Add(employeeAdvances)
	unexplainedDifference := expectedBalance.Sub(accountedTotal)

	return MoneyAnalysis{
		Month:                 monthStart.Format("2006-01"),
		Income:                income,
		Expense:               expense,
		SupplierCashPayments:  supplierCashPayments,
		ExpectedBalance:       expectedBalance,
		CashBalance:           cashBalance,
		BankBalance:           bankBalance,
		PendingPOS:            pendingPOS,
		CustomerReceivables:   customerReceivables,
		SupplierReceivables:   supplierReceivables,
		EmployeeAdvances:      employeeAdvances,
		AccountedTotal:        accountedTotal,
		UnexplainedDifference: unexplainedDifference,
		Breakdown: []MoneyAnalysisBreakdownItem{
			{Label: "Kasa", Amount: cashBalance},
			{Label: "Banka", Amount: bankBalance},
			{Label: "Bekleyen POS", Amount: pendingPOS},
			{Label: "Veresiye / Cari Alacaklar", Amount: customerReceivables},
			{Label: "Firmalardan Alacaklar", Amount: supplierReceivables},
			{Label: "Personel Avansları", Amount: employeeAdvances},
			{Label: "Bulunamayan Tutar", Amount: unexplainedDifference},
		},
	}, nil
}

func moneyAnalysisIncome(db *gorm.DB, start, end time.Time) (decimal.Decimal, error) {
	cashReportIncome, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(cash_amount + pos_amount + qr_amount), 0)::text
		FROM daily_cash_reports
		WHERE report_date >= ? AND report_date < ?
	`, start, end)
	if err != nil {
		return decimal.Zero, err
	}
	manualIncome, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM income_entries
		WHERE income_date >= ? AND income_date < ?
	`, start, end)
	if err != nil {
		return decimal.Zero, err
	}
	return cashReportIncome.Add(manualIncome), nil
}

func moneyAnalysisExpense(db *gorm.DB, start, end time.Time) (decimal.Decimal, decimal.Decimal, error) {
	total, supplierCashPayments, err := DailyCashOutflowBetween(db, start, end)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	queries := []struct {
		query string
	}{
		{`SELECT COALESCE(SUM(amount), 0)::text FROM employee_transactions WHERE type IN ('payment', 'advance') AND transaction_date >= ? AND transaction_date < ?`},
		{`SELECT COALESCE(SUM(amount), 0)::text FROM financial_debt_payments WHERE payment_date >= ? AND payment_date < ?`},
	}

	for _, item := range queries {
		amount, err := decimalFromQuery(db, item.query, start, end)
		if err != nil {
			return decimal.Zero, decimal.Zero, err
		}
		total = total.Add(amount)
	}
	return total, supplierCashPayments, nil
}

func moneyAnalysisWalletBalance(db *gorm.DB) (decimal.Decimal, error) {
	if !db.Migrator().HasTable("wallet_transactions") {
		return decimal.Zero, nil
	}
	return WalletCurrentBalance(db)
}

func moneyAnalysisBankBalance(db *gorm.DB) (decimal.Decimal, error) {
	if !db.Migrator().HasTable("bank_accounts") {
		return decimal.Zero, nil
	}
	return TotalBankBalance(db)
}

func moneyAnalysisCustomerReceivables(db *gorm.DB) (decimal.Decimal, error) {
	if !db.Migrator().HasTable("customer_transactions") {
		return decimal.Zero, nil
	}
	balance, err := TotalCustomerBalance(db)
	if err != nil {
		return decimal.Zero, err
	}
	if balance.IsNegative() {
		return decimal.Zero, nil
	}
	return balance, nil
}

func moneyAnalysisSupplierReceivables(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN balance < 0 THEN -balance ELSE 0 END), 0)::text
		FROM (
			SELECT COALESCE(SUM(CASE
				WHEN type IN ('invoice', 'purchase') THEN amount_try
				WHEN type IN ('payment', 'return') THEN -amount_try
				ELSE 0
			END), 0) AS balance
			FROM supplier_transactions
			GROUP BY supplier_id
		) supplier_balances
	`)
}

func moneyAnalysisEmployeeAdvances(db *gorm.DB, start, end time.Time) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM employee_transactions
		WHERE type = 'advance'
		AND transaction_date >= ? AND transaction_date < ?
	`, start, end)
}
