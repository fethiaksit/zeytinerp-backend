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
}

type EmployeeBalanceRow struct {
	EmployeeID uint            `json:"employee_id"`
	Name       string          `json:"name"`
	Phone      string          `json:"phone"`
	IsActive   bool            `json:"is_active"`
	Balance    decimal.Decimal `json:"balance"`
}

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
		SELECT COALESCE(SUM(CASE WHEN type = 'purchase' THEN amount WHEN type = 'payment' THEN -amount ELSE 0 END), 0)::text
		FROM supplier_transactions
		WHERE supplier_id = ?
	`, supplierID)
}

func TotalSupplierBalance(db *gorm.DB) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(CASE WHEN type = 'purchase' THEN amount WHEN type = 'payment' THEN -amount ELSE 0 END), 0)::text
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
	}
	err := db.Raw(`
		SELECT
			s.id AS supplier_id,
			s.name,
			s.phone,
			s.is_active,
			COALESCE(SUM(CASE
				WHEN st.type = 'purchase' THEN st.amount
				WHEN st.type = 'payment' THEN -st.amount
				ELSE 0
			END), 0)::text AS balance
		FROM suppliers s
		LEFT JOIN supplier_transactions st ON st.supplier_id = s.id
		GROUP BY s.id, s.name, s.phone, s.is_active
		ORDER BY COALESCE(SUM(CASE
			WHEN st.type = 'purchase' THEN st.amount
			WHEN st.type = 'payment' THEN -st.amount
			ELSE 0
		END), 0) DESC, s.name ASC
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
		result = append(result, SupplierBalanceRow{
			SupplierID: row.SupplierID,
			Name:       row.Name,
			Phone:      row.Phone,
			IsActive:   row.IsActive,
			Balance:    balance,
		})
	}
	return result, nil
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
