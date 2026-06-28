package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type FinanceCenterSummary struct {
	CashBalance          decimal.Decimal             `json:"cash_balance"`
	BankBalance          decimal.Decimal             `json:"bank_balance"`
	TotalMoney           decimal.Decimal             `json:"total_money"`
	SupplierDebts        decimal.Decimal             `json:"supplier_debts"`
	FinancialDebts       decimal.Decimal             `json:"financial_debts"`
	PersonnelDebts       decimal.Decimal             `json:"personnel_debts"`
	TotalDebts           decimal.Decimal             `json:"total_debts"`
	NetWorth             decimal.Decimal             `json:"net_worth"`
	MonthlyRevenue       decimal.Decimal             `json:"monthly_revenue"`
	MonthlyCollected     decimal.Decimal             `json:"monthly_collected"`
	MonthlyExpense       decimal.Decimal             `json:"monthly_expense"`
	MonthlyNetCashflow   decimal.Decimal             `json:"monthly_net_cashflow"`
	SupplierDebtTRY      decimal.Decimal             `json:"supplier_debt_try"`
	SupplierDebtUSD      decimal.Decimal             `json:"supplier_debt_usd"`
	SupplierDebtEUR      decimal.Decimal             `json:"supplier_debt_eur"`
	MonthlyDetails       FinanceCenterMonthlyDetails `json:"monthly_details"`
	LegacyCashBalance    decimal.Decimal             `json:"kasa_nakit"`
	LegacyBankBalance    decimal.Decimal             `json:"banka_bakiyesi"`
	LegacyTotalAssets    decimal.Decimal             `json:"toplam_varlik"`
	LegacySupplierDebts  decimal.Decimal             `json:"firma_borclari"`
	LegacyEmployeeDebts  decimal.Decimal             `json:"personel_borclari"`
	LegacyFinancialDebts decimal.Decimal             `json:"finans_borclari"`
	LegacyTotalDebt      decimal.Decimal             `json:"toplam_borc"`
	LegacyNetPosition    decimal.Decimal             `json:"net_durum"`
}

type FinanceCenterMonthlyDetails struct {
	Month                  string          `json:"month"`
	CashSales              decimal.Decimal `json:"cash_sales"`
	POSSales               decimal.Decimal `json:"pos_sales"`
	QRSales                decimal.Decimal `json:"qr_sales"`
	CreditSales            decimal.Decimal `json:"credit_sales"`
	IncomeEntrySales       decimal.Decimal `json:"income_entry_sales"`
	CreditCollections      decimal.Decimal `json:"credit_collections"`
	IncomeEntriesCollected decimal.Decimal `json:"income_entries_collected"`
	ExpenseEntries         decimal.Decimal `json:"expense_entries"`
	SupplierPayments       decimal.Decimal `json:"supplier_payments"`
	PersonnelPayments      decimal.Decimal `json:"personnel_payments"`
	PersonnelAdvances      decimal.Decimal `json:"personnel_advances"`
	FinancialDebtPayments  decimal.Decimal `json:"financial_debt_payments"`
}

type FinanceCenterHistory struct {
	Date          string          `json:"date"`
	CashBalance   decimal.Decimal `json:"kasa"`
	BankBalance   decimal.Decimal `json:"banka"`
	TotalMoney    decimal.Decimal `json:"toplam_para"`
	SupplierDebt  decimal.Decimal `json:"firma_borcu"`
	EmployeeDebt  decimal.Decimal `json:"personel_borcu"`
	FinancialDebt decimal.Decimal `json:"finans_borcu"`
	TotalDebt     decimal.Decimal `json:"toplam_borc"`
	NetPosition   decimal.Decimal `json:"net_durum"`
}

type FinanceCenterDifferenceTransaction struct {
	Source          string          `json:"source"`
	TransactionDate string          `json:"transaction_date"`
	TransactionType string          `json:"transaction_type"`
	Title           string          `json:"title"`
	Amount          decimal.Decimal `json:"amount"`
	Effect          decimal.Decimal `json:"effect"`
}

type FinanceCenterMoneyFlow struct {
	StartDate              string                               `json:"start_date"`
	EndDate                string                               `json:"end_date"`
	TotalIncome            decimal.Decimal                      `json:"toplam_gelir"`
	TotalExpense           decimal.Decimal                      `json:"toplam_gider"`
	ExpenseEntries         decimal.Decimal                      `json:"gider_kayitlari"`
	SupplierPayments       decimal.Decimal                      `json:"firma_odemeleri"`
	EmployeePayments       decimal.Decimal                      `json:"personel_odemeleri"`
	EmployeeAdvances       decimal.Decimal                      `json:"personel_avanslari"`
	FinancialPayments      decimal.Decimal                      `json:"finans_odemeleri"`
	OpeningBalance         decimal.Decimal                      `json:"acilis_bakiyesi"`
	ExpectedBalance        decimal.Decimal                      `json:"beklenen_bakiye"`
	ActualBalance          decimal.Decimal                      `json:"gercek_bakiye"`
	Difference             decimal.Decimal                      `json:"fark"`
	DifferenceTransactions []FinanceCenterDifferenceTransaction `json:"fark_islemleri"`
}

type FinanceCenterDebtDistribution struct {
	Suppliers       decimal.Decimal `json:"firma"`
	Employees       decimal.Decimal `json:"personel"`
	Loans           decimal.Decimal `json:"kredi"`
	CreditCards     decimal.Decimal `json:"kredi_karti"`
	OtherFinancials decimal.Decimal `json:"diger_finans_borclari"`
	TotalDebt       decimal.Decimal `json:"toplam_borc"`
}

type FinanceCenterCashflowMonth struct {
	Month   string          `json:"month"`
	Income  decimal.Decimal `json:"gelir"`
	Expense decimal.Decimal `json:"gider"`
	Net     decimal.Decimal `json:"net_kazanc"`
}

type FinanceCenterAlerts struct {
	OverdueDebts []models.FinancialDebtInstallment `json:"vadesi_gecmis_borclar"`
	DueIn7Days   []models.FinancialDebtInstallment `json:"yedi_gun_icin_odemeler"`
	DueIn30Days  []models.FinancialDebtInstallment `json:"otuz_gun_icin_odemeler"`
}

// FinanceCenterSummaryData returns the current liquid assets and outstanding debts.
func FinanceCenterSummaryData(db *gorm.DB) (FinanceCenterSummary, error) {
	cashBalance, err := moneyAnalysisWalletBalance(db)
	if err != nil {
		return FinanceCenterSummary{}, err
	}
	bankBalance, err := moneyAnalysisBankBalance(db)
	if err != nil {
		return FinanceCenterSummary{}, err
	}
	distribution, err := FinanceCenterDebtDistributionData(db)
	if err != nil {
		return FinanceCenterSummary{}, err
	}
	currencyTotals, err := SupplierCurrencyBalanceTotals(db, nil)
	if err != nil {
		return FinanceCenterSummary{}, err
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthlyDetails, monthlyRevenue, monthlyCollected, monthlyExpense, err := financeCenterMonthlyTotals(db, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		return FinanceCenterSummary{}, err
	}

	totalMoney := cashBalance.Add(bankBalance)
	financialDebts := distribution.Loans.Add(distribution.CreditCards).Add(distribution.OtherFinancials)
	totalDebts := distribution.Suppliers.Add(financialDebts).Add(distribution.Employees)
	return FinanceCenterSummary{
		CashBalance:          cashBalance,
		BankBalance:          bankBalance,
		TotalMoney:           totalMoney,
		SupplierDebts:        distribution.Suppliers,
		FinancialDebts:       financialDebts,
		PersonnelDebts:       distribution.Employees,
		TotalDebts:           totalDebts,
		NetWorth:             totalMoney.Sub(totalDebts),
		MonthlyRevenue:       monthlyRevenue,
		MonthlyCollected:     monthlyCollected,
		MonthlyExpense:       monthlyExpense,
		MonthlyNetCashflow:   monthlyCollected.Sub(monthlyExpense),
		SupplierDebtTRY:      currencyTotals.TRY,
		SupplierDebtUSD:      currencyTotals.USD,
		SupplierDebtEUR:      currencyTotals.EUR,
		MonthlyDetails:       monthlyDetails,
		LegacyCashBalance:    cashBalance,
		LegacyBankBalance:    bankBalance,
		LegacyTotalAssets:    totalMoney,
		LegacySupplierDebts:  distribution.Suppliers,
		LegacyEmployeeDebts:  distribution.Employees,
		LegacyFinancialDebts: financialDebts,
		LegacyTotalDebt:      totalDebts,
		LegacyNetPosition:    totalMoney.Sub(totalDebts),
	}, nil
}

func financeCenterMonthlyTotals(db *gorm.DB, start, end time.Time) (FinanceCenterMonthlyDetails, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	var dailyRows struct {
		CashSales         string
		POSSales          string
		QRSales           string
		CreditSales       string
		CreditCollections string
	}
	if err := db.Raw(`
		SELECT
			COALESCE(SUM(cash_amount), 0)::text AS cash_sales,
			COALESCE(SUM(pos_amount), 0)::text AS pos_sales,
			COALESCE(SUM(qr_amount), 0)::text AS qr_sales,
			COALESCE(SUM(credit_given), 0)::text AS credit_sales,
			COALESCE(SUM(credit_collection), 0)::text AS credit_collections
		FROM daily_cash_reports
		WHERE report_date >= ? AND report_date < ?
	`, start, end).Scan(&dailyRows).Error; err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	var incomeRows struct {
		SalesIncome string
		AllIncome   string
	}
	if err := db.Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN category IN ('market_satis', 'tup_satis') THEN amount ELSE 0 END), 0)::text AS sales_income,
			COALESCE(SUM(amount), 0)::text AS all_income
		FROM income_entries
		WHERE income_date >= ? AND income_date < ?
	`, start, end).Scan(&incomeRows).Error; err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	cashSales, err := decimal.NewFromString(dailyRows.CashSales)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	posSales, err := decimal.NewFromString(dailyRows.POSSales)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	qrSales, err := decimal.NewFromString(dailyRows.QRSales)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	creditSales, err := decimal.NewFromString(dailyRows.CreditSales)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	creditCollections, err := decimal.NewFromString(dailyRows.CreditCollections)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	incomeEntrySales, err := decimal.NewFromString(incomeRows.SalesIncome)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	allIncomeEntries, err := decimal.NewFromString(incomeRows.AllIncome)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	dailyCashOutflow, supplierPayments, err := DailyCashOutflowBetween(db, start, end)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	expenseEntries := dailyCashOutflow.Sub(supplierPayments)
	personnelPayments, err := decimalFromQuery(db, `SELECT COALESCE(SUM(amount), 0)::text FROM employee_transactions WHERE type = 'payment' AND transaction_date >= ? AND transaction_date < ?`, start, end)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	personnelAdvances, err := decimalFromQuery(db, `SELECT COALESCE(SUM(amount), 0)::text FROM employee_transactions WHERE type = 'advance' AND transaction_date >= ? AND transaction_date < ?`, start, end)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	financialPayments, err := decimalFromQuery(db, `SELECT COALESCE(SUM(amount), 0)::text FROM financial_debt_payments WHERE payment_date >= ? AND payment_date < ?`, start, end)
	if err != nil {
		return FinanceCenterMonthlyDetails{}, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	monthlyRevenue := cashSales.Add(posSales).Add(qrSales).Add(creditSales).Add(incomeEntrySales)
	monthlyCollected := cashSales.Add(posSales).Add(qrSales).Add(creditCollections).Add(allIncomeEntries)
	monthlyExpense := expenseEntries.Add(supplierPayments).Add(personnelPayments).Add(personnelAdvances).Add(financialPayments)
	return FinanceCenterMonthlyDetails{
		Month:                  start.Format("2006-01"),
		CashSales:              cashSales,
		POSSales:               posSales,
		QRSales:                qrSales,
		CreditSales:            creditSales,
		IncomeEntrySales:       incomeEntrySales,
		CreditCollections:      creditCollections,
		IncomeEntriesCollected: allIncomeEntries,
		ExpenseEntries:         expenseEntries,
		SupplierPayments:       supplierPayments,
		PersonnelPayments:      personnelPayments,
		PersonnelAdvances:      personnelAdvances,
		FinancialDebtPayments:  financialPayments,
	}, monthlyRevenue, monthlyCollected, monthlyExpense, nil
}

// FinanceCenterHistoryAt rebuilds cash and bank positions from transactions up
// to the requested day, rather than using the current account balance.
func FinanceCenterHistoryAt(db *gorm.DB, date time.Time) (FinanceCenterHistory, error) {
	cashBalance, err := financeCenterWalletBalanceAt(db, date)
	if err != nil {
		return FinanceCenterHistory{}, err
	}
	bankBalance, err := financeCenterBankBalanceAt(db, date)
	if err != nil {
		return FinanceCenterHistory{}, err
	}
	snapshot, err := DebtSnapshotAt(db, date)
	if err != nil {
		return FinanceCenterHistory{}, err
	}

	supplierDebt, employeeDebt, financialDebt := positiveSnapshotDebts(snapshot)
	totalMoney := cashBalance.Add(bankBalance)
	totalDebt := supplierDebt.Add(employeeDebt).Add(financialDebt)
	return FinanceCenterHistory{
		Date:          date.Format("2006-01-02"),
		CashBalance:   cashBalance,
		BankBalance:   bankBalance,
		TotalMoney:    totalMoney,
		SupplierDebt:  supplierDebt,
		EmployeeDebt:  employeeDebt,
		FinancialDebt: financialDebt,
		TotalDebt:     totalDebt,
		NetPosition:   totalMoney.Sub(totalDebt),
	}, nil
}

func FinanceCenterMoneyFlowData(db *gorm.DB, start, end time.Time) (FinanceCenterMoneyFlow, error) {
	income, err := moneyAnalysisIncome(db, start, end)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	dailyCashOutflow, supplierPayments, err := DailyCashOutflowBetween(db, start, end)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	expenseEntries := dailyCashOutflow.Sub(supplierPayments)
	employeePayments, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM employee_transactions
		WHERE type = 'payment' AND transaction_date >= ? AND transaction_date < ?
	`, start, end)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	employeeAdvances, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM employee_transactions
		WHERE type = 'advance' AND transaction_date >= ? AND transaction_date < ?
	`, start, end)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	financialPayments, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM financial_debt_payments
		WHERE payment_date >= ? AND payment_date < ?
	`, start, end)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}

	totalExpense := expenseEntries.Add(supplierPayments).Add(employeePayments).Add(employeeAdvances).Add(financialPayments)
	openingDate := start.AddDate(0, 0, -1)
	openingCash, err := financeCenterWalletBalanceAt(db, openingDate)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	openingBank, err := financeCenterBankBalanceAt(db, openingDate)
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	actualCash, err := financeCenterWalletBalanceAt(db, end.AddDate(0, 0, -1))
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}
	actualBank, err := financeCenterBankBalanceAt(db, end.AddDate(0, 0, -1))
	if err != nil {
		return FinanceCenterMoneyFlow{}, err
	}

	openingBalance := openingCash.Add(openingBank)
	expectedBalance := openingBalance.Add(income).Sub(totalExpense)
	actualBalance := actualCash.Add(actualBank)
	difference := expectedBalance.Sub(actualBalance)
	differenceTransactions := make([]FinanceCenterDifferenceTransaction, 0)
	if !difference.IsZero() {
		differenceTransactions, err = financeCenterUnlinkedTransactions(db, start, end)
		if err != nil {
			return FinanceCenterMoneyFlow{}, err
		}
	}

	return FinanceCenterMoneyFlow{
		StartDate:              start.Format("2006-01-02"),
		EndDate:                end.AddDate(0, 0, -1).Format("2006-01-02"),
		TotalIncome:            income,
		TotalExpense:           totalExpense,
		ExpenseEntries:         expenseEntries,
		SupplierPayments:       supplierPayments,
		EmployeePayments:       employeePayments,
		EmployeeAdvances:       employeeAdvances,
		FinancialPayments:      financialPayments,
		OpeningBalance:         openingBalance,
		ExpectedBalance:        expectedBalance,
		ActualBalance:          actualBalance,
		Difference:             difference,
		DifferenceTransactions: differenceTransactions,
	}, nil
}

func FinanceCenterDebtDistributionData(db *gorm.DB) (FinanceCenterDebtDistribution, error) {
	supplierDebts, err := financeCenterPositiveSupplierDebt(db)
	if err != nil {
		return FinanceCenterDebtDistribution{}, err
	}
	employeeDebts, err := financeCenterPositiveEmployeeDebt(db)
	if err != nil {
		return FinanceCenterDebtDistribution{}, err
	}
	financialRows, err := financeCenterCurrentFinancialDebts(db)
	if err != nil {
		return FinanceCenterDebtDistribution{}, err
	}

	loans := decimal.Zero
	creditCards := decimal.Zero
	otherFinancials := decimal.Zero
	for _, debt := range financialRows {
		remaining := positiveDecimalAmount(debt.RemainingAmount)
		switch debt.DebtType {
		case "bank_loan":
			loans = loans.Add(remaining)
		case "credit_card":
			creditCards = creditCards.Add(remaining)
		default:
			otherFinancials = otherFinancials.Add(remaining)
		}
	}
	totalDebt := supplierDebts.Add(employeeDebts).Add(loans).Add(creditCards).Add(otherFinancials)
	return FinanceCenterDebtDistribution{
		Suppliers:       supplierDebts,
		Employees:       employeeDebts,
		Loans:           loans,
		CreditCards:     creditCards,
		OtherFinancials: otherFinancials,
		TotalDebt:       totalDebt,
	}, nil
}

type financeCenterFinancialDebtRow struct {
	DebtType        string
	RemainingAmount decimal.Decimal
}

// financeCenterCurrentFinancialDebts keeps financial debts without an
// installment plan visible by using total_amount minus all payments.
func financeCenterCurrentFinancialDebts(db *gorm.DB) ([]financeCenterFinancialDebtRow, error) {
	var rows []struct {
		DebtType        string
		RemainingAmount string
	}
	if err := db.Raw(`
		SELECT
			fd.debt_type,
			(
				CASE WHEN EXISTS (
					SELECT 1 FROM financial_debt_installments i
					WHERE i.financial_debt_id = fd.id
				) THEN COALESCE((
					SELECT SUM(i.amount) FROM financial_debt_installments i
					WHERE i.financial_debt_id = fd.id
				), 0)
				ELSE fd.total_amount
				END
				- COALESCE((
					SELECT SUM(p.amount) FROM financial_debt_payments p
					WHERE p.financial_debt_id = fd.id
				), 0)
			)::text AS remaining_amount
		FROM financial_debts fd
		WHERE fd.status = 'active'
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]financeCenterFinancialDebtRow, 0, len(rows))
	for _, row := range rows {
		remainingAmount, err := decimal.NewFromString(row.RemainingAmount)
		if err != nil {
			return nil, err
		}
		result = append(result, financeCenterFinancialDebtRow{DebtType: row.DebtType, RemainingAmount: remainingAmount})
	}
	return result, nil
}

func FinanceCenterCashflowData(db *gorm.DB, until time.Time, monthCount int) ([]FinanceCenterCashflowMonth, error) {
	if monthCount < 1 {
		return []FinanceCenterCashflowMonth{}, nil
	}
	monthStart := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, until.Location()).AddDate(0, -(monthCount - 1), 0)
	monthEnd := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, until.Location()).AddDate(0, 1, 0)
	var rows []struct {
		Month   string
		Income  string
		Expense string
	}
	err := db.Raw(`
		WITH months AS (
			SELECT generate_series(?::date, (?::date - INTERVAL '1 month'), INTERVAL '1 month')::date AS month_start
		), income_rows AS (
			SELECT date_trunc('month', report_date)::date AS month_start, SUM(cash_amount + pos_amount + qr_amount) AS amount
			FROM daily_cash_reports
			WHERE report_date >= ? AND report_date < ?
			GROUP BY 1
			UNION ALL
			SELECT date_trunc('month', income_date)::date, SUM(amount)
			FROM income_entries
			WHERE income_date >= ? AND income_date < ?
			GROUP BY 1
		), expense_rows AS (
			SELECT date_trunc('month', expense_date)::date AS month_start, SUM(amount) AS amount
			FROM expenses WHERE expense_date >= ? AND expense_date < ? GROUP BY 1
			UNION ALL
			SELECT date_trunc('month', transaction_date)::date, SUM(amount_try)
			FROM supplier_transactions
			WHERE type = 'payment'
				AND LOWER(BTRIM(payment_method)) IN ('cash', 'nakit')
				AND transaction_date >= ? AND transaction_date < ?
			GROUP BY 1
			UNION ALL
			SELECT date_trunc('month', transaction_date)::date, SUM(amount)
			FROM employee_transactions WHERE type IN ('payment', 'advance') AND transaction_date >= ? AND transaction_date < ? GROUP BY 1
			UNION ALL
			SELECT date_trunc('month', payment_date)::date, SUM(amount)
			FROM financial_debt_payments WHERE payment_date >= ? AND payment_date < ? GROUP BY 1
		), monthly_income AS (
			SELECT month_start, SUM(amount) AS amount FROM income_rows GROUP BY month_start
		), monthly_expense AS (
			SELECT month_start, SUM(amount) AS amount FROM expense_rows GROUP BY month_start
		)
		SELECT
			to_char(m.month_start, 'YYYY-MM') AS month,
			COALESCE(i.amount, 0)::text AS income,
			COALESCE(e.amount, 0)::text AS expense
		FROM months m
		LEFT JOIN monthly_income i ON i.month_start = m.month_start
		LEFT JOIN monthly_expense e ON e.month_start = m.month_start
		ORDER BY m.month_start ASC
	`, monthStart, monthEnd, monthStart, monthEnd, monthStart, monthEnd, monthStart, monthEnd, monthStart, monthEnd, monthStart, monthEnd, monthStart, monthEnd).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]FinanceCenterCashflowMonth, 0, len(rows))
	for _, row := range rows {
		income, err := decimal.NewFromString(row.Income)
		if err != nil {
			return nil, err
		}
		expense, err := decimal.NewFromString(row.Expense)
		if err != nil {
			return nil, err
		}
		result = append(result, FinanceCenterCashflowMonth{Month: row.Month, Income: income, Expense: expense, Net: income.Sub(expense)})
	}
	return result, nil
}

func FinanceCenterAlertsData(db *gorm.DB) (FinanceCenterAlerts, error) {
	alerts, err := FinancialAlertsData(db)
	if err != nil {
		return FinanceCenterAlerts{}, err
	}
	return FinanceCenterAlerts{
		OverdueDebts: alerts.OverdueInstallments,
		DueIn7Days:   alerts.DueIn7Days,
		DueIn30Days:  alerts.DueIn30Days,
	}, nil
}

func financeCenterWalletBalanceAt(db *gorm.DB, date time.Time) (decimal.Decimal, error) {
	if !db.Migrator().HasTable("wallet_transactions") {
		return decimal.Zero, nil
	}
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE
			WHEN transaction_type IN ('opening_balance', 'cash_income', 'cash_sale', 'pos_income', 'bank_income', 'cash_deposit') THEN ABS(amount)
			WHEN transaction_type IN ('payment', 'expense', 'cash_withdraw') THEN -ABS(amount)
			WHEN transaction_type = 'correction' THEN amount
			ELSE 0
		END), 0)::text
		FROM wallet_transactions WHERE transaction_date <= ?
	`, date)
}

func financeCenterBankBalanceAt(db *gorm.DB, date time.Time) (decimal.Decimal, error) {
	if !db.Migrator().HasTable("bank_transactions") {
		return decimal.Zero, nil
	}
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE
			WHEN transaction_type IN ('opening_balance', 'cash_deposit', 'pos_income', 'bank_income', 'transfer_in') THEN ABS(amount)
			WHEN transaction_type IN ('payment', 'expense', 'transfer_out') THEN -ABS(amount)
			WHEN transaction_type = 'correction' THEN amount
			ELSE 0
		END), 0)::text
		FROM bank_transactions WHERE transaction_date <= ?
	`, date)
}

func financeCenterPositiveSupplierDebt(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN balance > 0 THEN balance ELSE 0 END), 0)::text
		FROM (
			SELECT COALESCE(SUM(CASE
				WHEN type IN ('invoice', 'purchase') THEN amount_try
				WHEN type IN ('payment', 'return') THEN -amount_try
				ELSE 0
			END), 0) AS balance
			FROM supplier_transactions GROUP BY supplier_id
		) supplier_balances
	`)
}

func financeCenterPositiveEmployeeDebt(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN balance > 0 THEN balance ELSE 0 END), 0)::text
		FROM (
			SELECT COALESCE(SUM(CASE
				WHEN et.type = 'work' THEN et.work_days * e.daily_wage
				WHEN et.type IN ('payment', 'advance') THEN -et.amount
				ELSE 0
			END), 0) AS balance
			FROM employees e
			LEFT JOIN employee_transactions et ON et.employee_id = e.id
			GROUP BY e.id
		) employee_balances
	`)
}

func positiveSnapshotDebts(snapshot DebtSnapshot) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	suppliers := decimal.Zero
	for _, row := range snapshot.Suppliers {
		suppliers = suppliers.Add(positiveDecimalAmount(row.Debt))
	}
	employees := decimal.Zero
	for _, row := range snapshot.Employees {
		employees = employees.Add(positiveDecimalAmount(row.Debt))
	}
	financials := decimal.Zero
	for _, row := range snapshot.FinancialDebts {
		financials = financials.Add(positiveDecimalAmount(row.RemainingAmount))
	}
	return suppliers, employees, financials
}

func positiveDecimalAmount(value decimal.Decimal) decimal.Decimal {
	if value.IsNegative() {
		return decimal.Zero
	}
	return value
}

func financeCenterUnlinkedTransactions(db *gorm.DB, start, end time.Time) ([]FinanceCenterDifferenceTransaction, error) {
	rows := make([]struct {
		Source          string
		TransactionDate time.Time
		TransactionType string
		Title           string
		Amount          string
		Effect          string
	}, 0)
	queries := make([]string, 0, 2)
	if db.Migrator().HasTable("wallet_transactions") {
		queries = append(queries, `
			SELECT 'wallet' AS source, transaction_date, transaction_type, title, amount::text AS amount,
				(CASE
					WHEN transaction_type IN ('opening_balance', 'cash_income', 'cash_sale', 'pos_income', 'bank_income', 'cash_deposit') THEN ABS(amount)
					WHEN transaction_type IN ('payment', 'expense', 'cash_withdraw') THEN -ABS(amount)
					WHEN transaction_type = 'correction' THEN amount ELSE 0
				END)::text AS effect
			FROM wallet_transactions
			WHERE transaction_date >= ? AND transaction_date < ?
			AND COALESCE(related_type, '') = '' AND related_id IS NULL`)
	}
	if db.Migrator().HasTable("bank_transactions") {
		queries = append(queries, `
			SELECT 'bank' AS source, transaction_date, transaction_type, title, amount::text AS amount,
				(CASE
					WHEN transaction_type IN ('opening_balance', 'cash_deposit', 'pos_income', 'bank_income', 'transfer_in') THEN ABS(amount)
					WHEN transaction_type IN ('payment', 'expense', 'transfer_out') THEN -ABS(amount)
					WHEN transaction_type = 'correction' THEN amount ELSE 0
				END)::text AS effect
			FROM bank_transactions
			WHERE transaction_date >= ? AND transaction_date < ?
			AND COALESCE(related_type, '') = '' AND related_id IS NULL`)
	}
	if len(queries) == 0 {
		return []FinanceCenterDifferenceTransaction{}, nil
	}

	query := "SELECT * FROM (" + queries[0]
	args := []interface{}{start, end}
	for _, item := range queries[1:] {
		query += " UNION ALL " + item
		args = append(args, start, end)
	}
	query += ") unlinked_transactions ORDER BY transaction_date DESC, source ASC"
	if err := db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]FinanceCenterDifferenceTransaction, 0, len(rows))
	for _, row := range rows {
		amount, err := decimal.NewFromString(row.Amount)
		if err != nil {
			return nil, err
		}
		effect, err := decimal.NewFromString(row.Effect)
		if err != nil {
			return nil, err
		}
		result = append(result, FinanceCenterDifferenceTransaction{
			Source:          row.Source,
			TransactionDate: row.TransactionDate.Format("2006-01-02"),
			TransactionType: row.TransactionType,
			Title:           row.Title,
			Amount:          amount,
			Effect:          effect,
		})
	}
	return result, nil
}
