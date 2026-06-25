package services

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SupplierBalanceRow struct {
	SupplierID uint            `json:"supplier_id"`
	Name       string          `json:"name"`
	Phone      string          `json:"phone"`
	IsActive   bool            `json:"is_active"`
	Balance    decimal.Decimal `json:"balance"`
	TRYBalance decimal.Decimal `json:"try_balance"`
	USDBalance decimal.Decimal `json:"usd_balance"`
	EURBalance decimal.Decimal `json:"eur_balance"`
}

type SupplierCurrencyTotals struct {
	TRY      decimal.Decimal `json:"try"`
	USD      decimal.Decimal `json:"usd"`
	EUR      decimal.Decimal `json:"eur"`
	TotalTRY decimal.Decimal `json:"total_try"`
}

type EmployeeBalanceRow struct {
	EmployeeID uint            `json:"employee_id"`
	Name       string          `json:"name"`
	Phone      string          `json:"phone"`
	IsActive   bool            `json:"is_active"`
	Balance    decimal.Decimal `json:"balance"`
}

const supplierBalanceCase = `
	CASE
		WHEN type IN ('invoice', 'purchase') THEN amount_try
		WHEN type IN ('payment', 'return') THEN -amount_try
		ELSE 0
	END
`

func decimalFromQuery(db *gorm.DB, query string, args ...interface{}) (decimal.Decimal, error) {
	var value string
	if err := db.Raw(query, args...).Scan(&value).Error; err != nil {
		return decimal.Zero, err
	}
	if value == "" {
		return decimal.Zero, nil
	}
	result, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, err
	}
	return result, nil
}

func SupplierBalance(db *gorm.DB, supplierID uint) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(`+supplierBalanceCase+`), 0)::text
		FROM supplier_transactions
		WHERE supplier_id = ?
	`, supplierID)
}

func TotalSupplierBalance(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(`+supplierBalanceCase+`), 0)::text
		FROM supplier_transactions
	`)
}

func SupplierBalances(db *gorm.DB) ([]SupplierBalanceRow, error) {
	var rows []struct {
		SupplierID uint
		Name       string
		Phone      string
		IsActive   bool
		Balance    string
		TRYBalance string
		USDBalance string
		EURBalance string
	}
	err := db.Raw(`
		SELECT
			s.id AS supplier_id,
			s.name,
			s.phone,
			s.is_active,
			COALESCE(SUM(CASE WHEN st.type IN ('invoice', 'purchase') THEN st.amount_try WHEN st.type IN ('payment', 'return') THEN -st.amount_try ELSE 0 END), 0)::text AS balance,
			COALESCE(SUM(CASE WHEN st.currency = 'TRY' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'TRY' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS try_balance,
			COALESCE(SUM(CASE WHEN st.currency = 'USD' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'USD' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS usd_balance,
			COALESCE(SUM(CASE WHEN st.currency = 'EUR' AND st.type IN ('invoice', 'purchase') THEN st.amount_original WHEN st.currency = 'EUR' AND st.type IN ('payment', 'return') THEN -st.amount_original ELSE 0 END), 0)::text AS eur_balance
		FROM suppliers s
		LEFT JOIN supplier_transactions st ON st.supplier_id = s.id
		GROUP BY s.id, s.name, s.phone, s.is_active
		ORDER BY COALESCE(SUM(CASE WHEN st.type IN ('invoice', 'purchase') THEN st.amount_try WHEN st.type IN ('payment', 'return') THEN -st.amount_try ELSE 0 END), 0) DESC, s.name ASC
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]SupplierBalanceRow, 0, len(rows))
	for _, row := range rows {
		balance, err := decimal.NewFromString(row.Balance)
		if err != nil {
			return nil, err
		}
		tryBalance, err := decimal.NewFromString(row.TRYBalance)
		if err != nil {
			return nil, err
		}
		usdBalance, err := decimal.NewFromString(row.USDBalance)
		if err != nil {
			return nil, err
		}
		eurBalance, err := decimal.NewFromString(row.EURBalance)
		if err != nil {
			return nil, err
		}
		result = append(result, SupplierBalanceRow{
			SupplierID: row.SupplierID,
			Name:       row.Name,
			Phone:      row.Phone,
			IsActive:   row.IsActive,
			Balance:    balance,
			TRYBalance: tryBalance,
			USDBalance: usdBalance,
			EURBalance: eurBalance,
		})
	}
	return result, nil
}

func SupplierCurrencyBalanceTotals(db *gorm.DB, supplierID *uint) (SupplierCurrencyTotals, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN currency = 'TRY' AND type IN ('invoice', 'purchase') THEN amount_original WHEN currency = 'TRY' AND type IN ('payment', 'return') THEN -amount_original ELSE 0 END), 0)::text AS try_total,
			COALESCE(SUM(CASE WHEN currency = 'USD' AND type IN ('invoice', 'purchase') THEN amount_original WHEN currency = 'USD' AND type IN ('payment', 'return') THEN -amount_original ELSE 0 END), 0)::text AS usd_total,
			COALESCE(SUM(CASE WHEN currency = 'EUR' AND type IN ('invoice', 'purchase') THEN amount_original WHEN currency = 'EUR' AND type IN ('payment', 'return') THEN -amount_original ELSE 0 END), 0)::text AS eur_total,
			COALESCE(SUM(` + supplierBalanceCase + `), 0)::text AS total_try
		FROM supplier_transactions
	`
	args := []interface{}{}
	if supplierID != nil {
		query += " WHERE supplier_id = ?"
		args = append(args, *supplierID)
	}
	var row struct {
		TRYTotal string
		USDTotal string
		EURTotal string
		TotalTRY string
	}
	if err := db.Raw(query, args...).Scan(&row).Error; err != nil {
		return SupplierCurrencyTotals{}, err
	}
	tryTotal, err := decimal.NewFromString(row.TRYTotal)
	if err != nil {
		return SupplierCurrencyTotals{}, err
	}
	usdTotal, err := decimal.NewFromString(row.USDTotal)
	if err != nil {
		return SupplierCurrencyTotals{}, err
	}
	eurTotal, err := decimal.NewFromString(row.EURTotal)
	if err != nil {
		return SupplierCurrencyTotals{}, err
	}
	totalTRY, err := decimal.NewFromString(row.TotalTRY)
	if err != nil {
		return SupplierCurrencyTotals{}, err
	}
	return SupplierCurrencyTotals{TRY: tryTotal, USD: usdTotal, EUR: eurTotal, TotalTRY: totalTRY}, nil
}

func EmployeeBalance(db *gorm.DB, employeeID uint) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE
			WHEN et.type = 'work' THEN et.work_days * e.daily_wage
			WHEN et.type IN ('payment', 'advance') THEN -et.amount
			ELSE 0
		END), 0)::text
		FROM employee_transactions et
		JOIN employees e ON e.id = et.employee_id
		WHERE et.employee_id = ?
	`, employeeID)
}

func TotalEmployeeBalance(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE
			WHEN et.type = 'work' THEN et.work_days * e.daily_wage
			WHEN et.type IN ('payment', 'advance') THEN -et.amount
			ELSE 0
		END), 0)::text
		FROM employee_transactions et
		JOIN employees e ON e.id = et.employee_id
	`)
}

func EmployeeBalances(db *gorm.DB) ([]EmployeeBalanceRow, error) {
	var rows []struct {
		EmployeeID uint
		Name       string
		Phone      string
		IsActive   bool
		Balance    string
	}
	err := db.Raw(`
		SELECT
			e.id AS employee_id,
			e.name,
			e.phone,
			e.is_active,
			COALESCE(SUM(CASE
				WHEN et.type = 'work' THEN et.work_days * e.daily_wage
				WHEN et.type IN ('payment', 'advance') THEN -et.amount
				ELSE 0
			END), 0)::text AS balance
		FROM employees e
		LEFT JOIN employee_transactions et ON et.employee_id = e.id
		GROUP BY e.id, e.name, e.phone, e.is_active
		ORDER BY COALESCE(SUM(CASE
			WHEN et.type = 'work' THEN et.work_days * e.daily_wage
			WHEN et.type IN ('payment', 'advance') THEN -et.amount
			ELSE 0
		END), 0) DESC, e.name ASC
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]EmployeeBalanceRow, 0, len(rows))
	for _, row := range rows {
		balance, err := decimal.NewFromString(row.Balance)
		if err != nil {
			return nil, err
		}
		result = append(result, EmployeeBalanceRow{
			EmployeeID: row.EmployeeID,
			Name:       row.Name,
			Phone:      row.Phone,
			IsActive:   row.IsActive,
			Balance:    balance,
		})
	}
	return result, nil
}

func CustomerBalance(db *gorm.DB, customerID uint) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN type = 'debt' THEN amount WHEN type = 'payment' THEN -amount ELSE 0 END), 0)::text
		FROM customer_transactions
		WHERE customer_id = ?
	`, customerID)
}

func TotalCustomerBalance(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN type = 'debt' THEN amount WHEN type = 'payment' THEN -amount ELSE 0 END), 0)::text
		FROM customer_transactions
	`)
}

func ProductStock(db *gorm.DB, productID uint) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE
			WHEN type IN ('in', 'correction') THEN quantity
			WHEN type IN ('out', 'waste') THEN -quantity
			ELSE 0
		END), 0)::text
		FROM stock_movements
		WHERE product_id = ?
	`, productID)
}

func CriticalStockCount(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) FROM products p
		WHERE p.is_active = true
		AND (
			SELECT COALESCE(SUM(CASE
				WHEN sm.type IN ('in', 'correction') THEN sm.quantity
				WHEN sm.type IN ('out', 'waste') THEN -sm.quantity
				ELSE 0
			END), 0)
			FROM stock_movements sm
			WHERE sm.product_id = p.id
		) <= p.critical_stock
	`).Scan(&count).Error
	return count, err
}
