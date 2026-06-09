package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/services"
)

type DashboardHandler struct{ DB *gorm.DB }

type recentTransactionRow struct {
	Source          string          `json:"source"`
	ReferenceID     uint            `json:"reference_id"`
	TransactionDate time.Time       `json:"transaction_date"`
	Type            string          `json:"type"`
	Name            string          `json:"name"`
	Amount          decimal.Decimal `json:"amount"`
	Note            string          `json:"note"`
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler { return &DashboardHandler{DB: db} }

func (h *DashboardHandler) Get(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	data, err := h.dashboardData(today, today.AddDate(0, 0, 1), monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, data)
}

func (h *DashboardHandler) Monthly(c *gin.Context) {
	year, month, monthStart, monthEnd, valid := h.monthRange(c)
	if !valid {
		return
	}
	revenue, err := h.revenueBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	expense, err := h.expenseBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	cashRevenue, err := h.cashRevenueBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	incomeRevenue, err := h.incomeBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"year":                 year,
		"month":                month,
		"start_date":           monthStart.Format("2006-01-02"),
		"end_date":             monthEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		"month_revenue":        revenue,
		"daily_cash_revenue":   cashRevenue,
		"income_entries_total": incomeRevenue,
		"month_expense":        expense,
		"month_net":            revenue.Sub(expense),
	})
}

func (h *DashboardHandler) Reports(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	data, err := h.dashboardData(today, today.AddDate(0, 0, 1), monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, reportSummary(data))
}

func (h *DashboardHandler) ReportsMonthly(c *gin.Context) {
	year, month, monthStart, monthEnd, valid := h.monthRange(c)
	if !valid {
		return
	}
	revenue, err := h.revenueBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	expense, err := h.expenseBetween(monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	monthlyFinancialDue, err := services.MonthlyFinancialDue(h.DB, monthStart, monthEnd)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	overdueFinancialCount, err := services.OverdueFinancialInstallmentCount(h.DB)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"year":                    year,
		"month":                   month,
		"start_date":              monthStart.Format("2006-01-02"),
		"end_date":                monthEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		"month_revenue":           revenue,
		"month_expense":           expense,
		"month_net":               revenue.Sub(expense),
		"monthly_financial_due":   monthlyFinancialDue,
		"overdue_financial_count": overdueFinancialCount,
	})
}

func (h *DashboardHandler) monthRange(c *gin.Context) (int, int, time.Time, time.Time, bool) {
	year, err := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	if err != nil || year < 2000 || year > 2100 {
		fail(c, http.StatusBadRequest, "year is invalid")
		return 0, 0, time.Time{}, time.Time{}, false
	}
	month, err := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	if err != nil || month < 1 || month > 12 {
		fail(c, http.StatusBadRequest, "month is invalid")
		return 0, 0, time.Time{}, time.Time{}, false
	}
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return year, month, monthStart, monthStart.AddDate(0, 1, 0), true
}

func (h *DashboardHandler) dashboardData(todayStart, todayEnd, monthStart, monthEnd time.Time) (gin.H, error) {
	todayRevenue, err := h.revenueBetween(todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	todayExpense, err := h.expenseBetween(todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	monthRevenue, err := h.revenueBetween(monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	monthExpense, err := h.expenseBetween(monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	supplierDebt, err := services.TotalSupplierBalance(h.DB)
	if err != nil {
		return nil, err
	}
	employeeDebt, err := services.TotalEmployeeBalance(h.DB)
	if err != nil {
		return nil, err
	}
	financialDebt, err := services.TotalFinancialDebt(h.DB)
	if err != nil {
		return nil, err
	}
	walletBalance, err := services.WalletCurrentBalance(h.DB)
	if err != nil {
		return nil, err
	}
	totalBankBalance, err := services.TotalBankBalance(h.DB)
	if err != nil {
		return nil, err
	}
	monthlyFinancialDue, err := services.MonthlyFinancialDue(h.DB, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	overdueFinancialCount, err := services.OverdueFinancialInstallmentCount(h.DB)
	if err != nil {
		return nil, err
	}
	upcomingFinancialDue7Days, err := services.UpcomingFinancialDue(h.DB, todayStart, todayStart.AddDate(0, 0, 8))
	if err != nil {
		return nil, err
	}
	upcomingFinancialDue30Days, err := services.UpcomingFinancialDue(h.DB, todayStart, todayStart.AddDate(0, 0, 31))
	if err != nil {
		return nil, err
	}
	nearestFinancialInstallments, err := services.NearestFinancialInstallments(h.DB, todayStart, todayStart.AddDate(0, 0, 31), 10)
	if err != nil {
		return nil, err
	}
	totalCash, err := dashboardDecimal(h.DB, `
		SELECT COALESCE(SUM(cash_amount), 0)::text
		FROM daily_cash_reports
	`)
	if err != nil {
		return nil, err
	}
	totalPos, err := dashboardDecimal(h.DB, `
		SELECT COALESCE(SUM(pos_amount), 0)::text
		FROM daily_cash_reports
	`)
	if err != nil {
		return nil, err
	}
	topSupplierDebts, err := h.topSupplierDebts()
	if err != nil {
		return nil, err
	}
	recentTransactions, err := h.recentTransactions()
	if err != nil {
		return nil, err
	}

	return gin.H{
		"today_revenue":                  todayRevenue,
		"today_expense":                  todayExpense,
		"today_net":                      todayRevenue.Sub(todayExpense),
		"month_revenue":                  monthRevenue,
		"month_expense":                  monthExpense,
		"month_net":                      monthRevenue.Sub(monthExpense),
		"total_supplier_debt":            supplierDebt,
		"total_employee_debt":            employeeDebt,
		"total_financial_debt":           financialDebt,
		"wallet_balance":                 walletBalance,
		"total_bank_balance":             totalBankBalance,
		"monthly_financial_due":          monthlyFinancialDue,
		"overdue_financial_count":        overdueFinancialCount,
		"upcoming_financial_due_7_days":  upcomingFinancialDue7Days,
		"upcoming_financial_due_30_days": upcomingFinancialDue30Days,
		"nearest_financial_installments": nearestFinancialInstallments,
		"due_this_month":                 monthlyFinancialDue,
		"overdue_debt_count":             overdueFinancialCount,
		"nearest_due_debts":              nearestFinancialInstallments,
		"total_cash":                     totalCash,
		"total_pos":                      totalPos,
		"top_supplier_debts":             topSupplierDebts,
		"recent_transactions":            recentTransactions,
	}, nil
}

func reportSummary(data gin.H) gin.H {
	return gin.H{
		"today_revenue":           data["today_revenue"],
		"today_expense":           data["today_expense"],
		"today_net":               data["today_net"],
		"month_revenue":           data["month_revenue"],
		"month_expense":           data["month_expense"],
		"month_net":               data["month_net"],
		"total_supplier_debt":     data["total_supplier_debt"],
		"total_employee_debt":     data["total_employee_debt"],
		"total_financial_debt":    data["total_financial_debt"],
		"wallet_balance":          data["wallet_balance"],
		"total_bank_balance":      data["total_bank_balance"],
		"monthly_financial_due":   data["monthly_financial_due"],
		"overdue_financial_count": data["overdue_financial_count"],
	}
}

func (h *DashboardHandler) revenueBetween(start, end time.Time) (decimal.Decimal, error) {
	cashRevenue, err := h.cashRevenueBetween(start, end)
	if err != nil {
		return decimal.Zero, err
	}
	incomeRevenue, err := h.incomeBetween(start, end)
	if err != nil {
		return decimal.Zero, err
	}
	return cashRevenue.Add(incomeRevenue), nil
}

func (h *DashboardHandler) cashRevenueBetween(start, end time.Time) (decimal.Decimal, error) {
	return dashboardDecimal(h.DB, `
		SELECT COALESCE(SUM(cash_amount + pos_amount + qr_amount), 0)::text
		FROM daily_cash_reports
		WHERE report_date >= ? AND report_date < ?
	`, start, end)
}

func (h *DashboardHandler) incomeBetween(start, end time.Time) (decimal.Decimal, error) {
	return dashboardDecimal(h.DB, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM income_entries
		WHERE income_date >= ? AND income_date < ?
	`, start, end)
}

func (h *DashboardHandler) expenseBetween(start, end time.Time) (decimal.Decimal, error) {
	return dashboardDecimal(h.DB, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM expenses
		WHERE expense_date >= ? AND expense_date < ?
	`, start, end)
}

func (h *DashboardHandler) topSupplierDebts() ([]services.SupplierBalanceRow, error) {
	balances, err := services.SupplierBalances(h.DB)
	if err != nil {
		return nil, err
	}
	result := make([]services.SupplierBalanceRow, 0, 5)
	for _, balance := range balances {
		if balance.Balance.LessThanOrEqual(decimal.Zero) {
			continue
		}
		result = append(result, balance)
		if len(result) == 5 {
			break
		}
	}
	return result, nil
}

func (h *DashboardHandler) recentTransactions() ([]recentTransactionRow, error) {
	var rows []struct {
		Source          string
		ReferenceID     uint
		TransactionDate time.Time
		Type            string
		Name            string
		Amount          string
		Note            string
	}
	if err := h.DB.Raw(`
		SELECT * FROM (
			SELECT 'supplier' AS source, st.id AS reference_id, st.transaction_date, st.type, s.name, st.amount::text AS amount, st.note
			FROM supplier_transactions st
			JOIN suppliers s ON s.id = st.supplier_id
			UNION ALL
			SELECT 'employee' AS source, et.id AS reference_id, et.transaction_date, et.type, e.name,
				(CASE WHEN et.type = 'work' THEN et.work_days * e.daily_wage ELSE et.amount END)::text AS amount,
				et.note
			FROM employee_transactions et
			JOIN employees e ON e.id = et.employee_id
			UNION ALL
			SELECT 'expense' AS source, ex.id AS reference_id, ex.expense_date AS transaction_date, ex.category AS type, 'Gider' AS name, ex.amount::text AS amount, ex.note
			FROM expenses ex
			UNION ALL
			SELECT 'income' AS source, ie.id AS reference_id, ie.income_date AS transaction_date, ie.category AS type, 'Gelir' AS name, ie.amount::text AS amount, ie.note
			FROM income_entries ie
		) recent
		ORDER BY transaction_date DESC, reference_id DESC
		LIMIT 10
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]recentTransactionRow, 0, len(rows))
	for _, row := range rows {
		amount, err := decimal.NewFromString(row.Amount)
		if err != nil {
			return nil, err
		}
		result = append(result, recentTransactionRow{
			Source:          row.Source,
			ReferenceID:     row.ReferenceID,
			TransactionDate: row.TransactionDate,
			Type:            row.Type,
			Name:            row.Name,
			Amount:          amount,
			Note:            row.Note,
		})
	}
	return result, nil
}

func dashboardDecimal(db *gorm.DB, query string, args ...interface{}) (decimal.Decimal, error) {
	var value string
	if err := db.Raw(query, args...).Scan(&value).Error; err != nil {
		return decimal.Zero, err
	}
	if value == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(value)
}
