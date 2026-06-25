package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type DebtSnapshotSupplier struct {
	SupplierID     uint                   `json:"supplier_id"`
	SupplierName   string                 `json:"supplier_name"`
	Debt           decimal.Decimal        `json:"debt"`
	CurrencyTotals SupplierCurrencyTotals `json:"currency_totals"`
}

type DebtSnapshotEmployee struct {
	EmployeeID   uint            `json:"employee_id"`
	EmployeeName string          `json:"employee_name"`
	Debt         decimal.Decimal `json:"debt"`
}

type DebtSnapshotFinancialDebt struct {
	ID              uint            `json:"id"`
	Title           string          `json:"title"`
	DebtType        string          `json:"debt_type"`
	Institution     string          `json:"institution"`
	RemainingAmount decimal.Decimal `json:"remaining_amount"`
}

type DebtSnapshot struct {
	Date               string                      `json:"date"`
	SupplierDebtTotal  decimal.Decimal             `json:"supplier_debt_total"`
	EmployeeDebtTotal  decimal.Decimal             `json:"employee_debt_total"`
	FinancialDebtTotal decimal.Decimal             `json:"financial_debt_total"`
	BankLoanTotal      decimal.Decimal             `json:"bank_loan_total"`
	CreditCardTotal    decimal.Decimal             `json:"credit_card_total"`
	TotalDebt          decimal.Decimal             `json:"total_debt"`
	Suppliers          []DebtSnapshotSupplier      `json:"suppliers"`
	FinancialDebts     []DebtSnapshotFinancialDebt `json:"financial_debts"`
	Employees          []DebtSnapshotEmployee      `json:"employees"`
}

func DebtSnapshotAt(db *gorm.DB, date time.Time) (DebtSnapshot, error) {
	suppliers, supplierTotal, err := debtSnapshotSuppliers(db, date)
	if err != nil {
		return DebtSnapshot{}, err
	}
	employees, employeeTotal, err := debtSnapshotEmployees(db, date)
	if err != nil {
		return DebtSnapshot{}, err
	}
	financialDebts, financialTotal, bankLoanTotal, creditCardTotal, err := debtSnapshotFinancialDebts(db, date)
	if err != nil {
		return DebtSnapshot{}, err
	}

	return DebtSnapshot{
		Date:               date.Format("2006-01-02"),
		SupplierDebtTotal:  supplierTotal,
		EmployeeDebtTotal:  employeeTotal,
		FinancialDebtTotal: financialTotal,
		BankLoanTotal:      bankLoanTotal,
		CreditCardTotal:    creditCardTotal,
		TotalDebt:          supplierTotal.Add(employeeTotal).Add(financialTotal),
		Suppliers:          suppliers,
		FinancialDebts:     financialDebts,
		Employees:          employees,
	}, nil
}

func debtSnapshotSuppliers(db *gorm.DB, date time.Time) ([]DebtSnapshotSupplier, decimal.Decimal, error) {
	var rows []struct {
		SupplierID   uint
		SupplierName string
		Debt         string
		TRYTotal     string
		USDTotal     string
		EURTotal     string
	}
	if err := db.Raw(`
		SELECT
			s.id AS supplier_id,
			s.name AS supplier_name,
			COALESCE(SUM(CASE
				WHEN st.type IN ('invoice', 'purchase') THEN st.amount_try
				WHEN st.type IN ('payment', 'return') THEN -st.amount_try
				ELSE 0 END), 0)::text AS debt,
			COALESCE(SUM(CASE WHEN st.currency = 'TRY' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'TRY' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS try_total,
			COALESCE(SUM(CASE WHEN st.currency = 'USD' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'USD' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS usd_total,
			COALESCE(SUM(CASE WHEN st.currency = 'EUR' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'EUR' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS eur_total
		FROM suppliers s
		LEFT JOIN supplier_transactions st
			ON st.supplier_id = s.id
			AND st.transaction_date <= ?
		GROUP BY s.id, s.name
		HAVING COALESCE(SUM(CASE
			WHEN st.type IN ('invoice', 'purchase') THEN st.amount_try
			WHEN st.type IN ('payment', 'return') THEN -st.amount_try
			ELSE 0
		END), 0) <> 0
		ORDER BY COALESCE(SUM(CASE
			WHEN st.type IN ('invoice', 'purchase') THEN st.amount_try
			WHEN st.type IN ('payment', 'return') THEN -st.amount_try
			ELSE 0
		END), 0) DESC, s.name ASC
	`, date).Scan(&rows).Error; err != nil {
		return nil, decimal.Zero, err
	}

	suppliers := make([]DebtSnapshotSupplier, 0, len(rows))
	total := decimal.Zero
	for _, row := range rows {
		debt, err := decimal.NewFromString(row.Debt)
		if err != nil {
			return nil, decimal.Zero, err
		}
		tryTotal, err := decimal.NewFromString(row.TRYTotal)
		if err != nil {
			return nil, decimal.Zero, err
		}
		usdTotal, err := decimal.NewFromString(row.USDTotal)
		if err != nil {
			return nil, decimal.Zero, err
		}
		eurTotal, err := decimal.NewFromString(row.EURTotal)
		if err != nil {
			return nil, decimal.Zero, err
		}
		suppliers = append(suppliers, DebtSnapshotSupplier{
			SupplierID:     row.SupplierID,
			SupplierName:   row.SupplierName,
			Debt:           debt,
			CurrencyTotals: SupplierCurrencyTotals{TRY: tryTotal, USD: usdTotal, EUR: eurTotal, TotalTRY: debt},
		})
		total = total.Add(debt)
	}
	return suppliers, total, nil
}

func debtSnapshotEmployees(db *gorm.DB, date time.Time) ([]DebtSnapshotEmployee, decimal.Decimal, error) {
	var rows []struct {
		EmployeeID   uint
		EmployeeName string
		Debt         string
	}
	if err := db.Raw(`
		SELECT
			e.id AS employee_id,
			e.name AS employee_name,
			COALESCE(SUM(CASE
				WHEN et.type = 'work' THEN et.work_days * e.daily_wage
				WHEN et.type IN ('payment', 'advance') THEN -et.amount
				ELSE 0
			END), 0)::text AS debt
		FROM employees e
		LEFT JOIN employee_transactions et
			ON et.employee_id = e.id
			AND et.transaction_date <= ?
		GROUP BY e.id, e.name
		HAVING COALESCE(SUM(CASE
			WHEN et.type = 'work' THEN et.work_days * e.daily_wage
			WHEN et.type IN ('payment', 'advance') THEN -et.amount
			ELSE 0
		END), 0) <> 0
		ORDER BY COALESCE(SUM(CASE
			WHEN et.type = 'work' THEN et.work_days * e.daily_wage
			WHEN et.type IN ('payment', 'advance') THEN -et.amount
			ELSE 0
		END), 0) DESC, e.name ASC
	`, date).Scan(&rows).Error; err != nil {
		return nil, decimal.Zero, err
	}

	employees := make([]DebtSnapshotEmployee, 0, len(rows))
	total := decimal.Zero
	for _, row := range rows {
		debt, err := decimal.NewFromString(row.Debt)
		if err != nil {
			return nil, decimal.Zero, err
		}
		employees = append(employees, DebtSnapshotEmployee{
			EmployeeID:   row.EmployeeID,
			EmployeeName: row.EmployeeName,
			Debt:         debt,
		})
		total = total.Add(debt)
	}
	return employees, total, nil
}

func debtSnapshotFinancialDebts(db *gorm.DB, date time.Time) ([]DebtSnapshotFinancialDebt, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	var rows []struct {
		ID              uint
		Title           string
		DebtType        string
		Institution     string
		RemainingAmount string
	}
	if err := db.Raw(`
		SELECT
			fd.id,
			fd.title,
			fd.debt_type,
			fd.institution_name AS institution,
			(
				CASE
					WHEN EXISTS (
						SELECT 1
						FROM financial_debt_installments i
						WHERE i.financial_debt_id = fd.id
					) THEN COALESCE((
						SELECT SUM(i.amount)
						FROM financial_debt_installments i
						WHERE i.financial_debt_id = fd.id
						AND i.due_date <= ?
					), 0)
					ELSE fd.total_amount
				END
				- COALESCE((
					SELECT SUM(p.amount)
					FROM financial_debt_payments p
					WHERE p.financial_debt_id = fd.id
					AND p.payment_date <= ?
				), 0)
			)::text AS remaining_amount
		FROM financial_debts fd
		WHERE fd.start_date <= ?
		AND (
			CASE
				WHEN EXISTS (
					SELECT 1
					FROM financial_debt_installments i
					WHERE i.financial_debt_id = fd.id
				) THEN COALESCE((
					SELECT SUM(i.amount)
					FROM financial_debt_installments i
					WHERE i.financial_debt_id = fd.id
					AND i.due_date <= ?
				), 0)
				ELSE fd.total_amount
			END
			- COALESCE((
				SELECT SUM(p.amount)
				FROM financial_debt_payments p
				WHERE p.financial_debt_id = fd.id
				AND p.payment_date <= ?
			), 0)
		) <> 0
		ORDER BY fd.end_date ASC, fd.id DESC
	`, date, date, date, date, date).Scan(&rows).Error; err != nil {
		return nil, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	debts := make([]DebtSnapshotFinancialDebt, 0, len(rows))
	total := decimal.Zero
	bankLoanTotal := decimal.Zero
	creditCardTotal := decimal.Zero
	for _, row := range rows {
		remainingAmount, err := decimal.NewFromString(row.RemainingAmount)
		if err != nil {
			return nil, decimal.Zero, decimal.Zero, decimal.Zero, err
		}
		debts = append(debts, DebtSnapshotFinancialDebt{
			ID:              row.ID,
			Title:           row.Title,
			DebtType:        row.DebtType,
			Institution:     row.Institution,
			RemainingAmount: remainingAmount,
		})
		total = total.Add(remainingAmount)
		switch row.DebtType {
		case "bank_loan":
			bankLoanTotal = bankLoanTotal.Add(remainingAmount)
		case "credit_card":
			creditCardTotal = creditCardTotal.Add(remainingAmount)
		}
	}
	return debts, total, bankLoanTotal, creditCardTotal, nil
}
