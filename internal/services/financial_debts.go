package services

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type FinancialDebtRow struct {
	ID               uint            `json:"id"`
	DebtType         string          `json:"debt_type"`
	InstitutionName  string          `json:"institution_name"`
	Title            string          `json:"title"`
	TotalAmount      decimal.Decimal `json:"total_amount"`
	InstallmentTotal decimal.Decimal `json:"installment_total"`
	PaidTotal        decimal.Decimal `json:"paid_total"`
	RemainingAmount  decimal.Decimal `json:"remaining_amount"`
	StartDate        time.Time       `json:"start_date"`
	EndDate          time.Time       `json:"end_date"`
	Status           string          `json:"status"`
	Note             string          `json:"note"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type FinancialDebtSummary struct {
	FinancialDebt    FinancialDebtRow                  `json:"financial_debt"`
	InstallmentTotal decimal.Decimal                   `json:"installment_total"`
	PaidTotal        decimal.Decimal                   `json:"paid_total"`
	RemainingAmount  decimal.Decimal                   `json:"remaining_amount"`
	Installments     []models.FinancialDebtInstallment `json:"installments"`
}

type FinancialAlerts struct {
	OverdueInstallments []models.FinancialDebtInstallment `json:"overdue_installments"`
	DueToday            []models.FinancialDebtInstallment `json:"due_today"`
	DueIn7Days          []models.FinancialDebtInstallment `json:"due_in_7_days"`
	DueIn30Days         []models.FinancialDebtInstallment `json:"due_in_30_days"`
}

func RefreshFinancialInstallmentStatuses(db *gorm.DB) error {
	var installments []models.FinancialDebtInstallment
	if err := db.Find(&installments).Error; err != nil {
		return err
	}
	for _, installment := range installments {
		if err := RecalculateFinancialInstallment(db, installment.ID); err != nil {
			return err
		}
	}
	return nil
}

func RecalculateFinancialInstallment(db *gorm.DB, installmentID uint) error {
	var installment models.FinancialDebtInstallment
	if err := db.First(&installment, installmentID).Error; err != nil {
		return err
	}

	paidAmount, err := decimalFromQuery(db, `
		SELECT COALESCE(SUM(amount), 0)::text
		FROM financial_debt_payments
		WHERE installment_id = ?
	`, installmentID)
	if err != nil {
		return err
	}

	status := FinancialInstallmentStatus(installment.DueDate, installment.Amount, paidAmount, time.Now())
	return db.Model(&models.FinancialDebtInstallment{}).
		Where("id = ?", installmentID).
		Updates(map[string]interface{}{"paid_amount": paidAmount, "status": status}).Error
}

func FinancialInstallmentStatus(dueDate time.Time, amount, paidAmount decimal.Decimal, now time.Time) string {
	if paidAmount.GreaterThanOrEqual(amount) {
		return "paid"
	}
	if dueDate.Before(startOfDay(now)) {
		return "overdue"
	}
	if paidAmount.GreaterThan(decimal.Zero) {
		return "partial_paid"
	}
	return "pending"
}

func FinancialDebtBalance(db *gorm.DB, debtID uint) (decimal.Decimal, error) {
	return decimalFromQuery(db, `
		SELECT (COALESCE(SUM(i.amount), 0) - COALESCE((
			SELECT SUM(p.amount)
			FROM financial_debt_payments p
			WHERE p.financial_debt_id = ?
		), 0))::text
		FROM financial_debt_installments i
		WHERE i.financial_debt_id = ?
	`, debtID, debtID)
}

func TotalFinancialDebt(db *gorm.DB) (decimal.Decimal, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return decimal.Zero, err
	}
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(balance), 0)::text
		FROM (
			SELECT fd.id, COALESCE(SUM(i.amount - i.paid_amount), 0) AS balance
			FROM financial_debts fd
			LEFT JOIN financial_debt_installments i ON i.financial_debt_id = fd.id
			WHERE fd.status = 'active'
			GROUP BY fd.id
		) debts
		WHERE balance > 0
	`)
}

func MonthlyFinancialDue(db *gorm.DB, start, end time.Time) (decimal.Decimal, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return decimal.Zero, err
	}
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(i.amount - i.paid_amount), 0)::text
		FROM financial_debt_installments i
		JOIN financial_debts fd ON fd.id = i.financial_debt_id
		WHERE fd.status = 'active'
		AND i.due_date >= ? AND i.due_date < ?
		AND i.status IN ('pending', 'partial_paid', 'overdue')
	`, start, end)
}

func OverdueFinancialInstallmentCount(db *gorm.DB) (int64, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return 0, err
	}
	var count int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM financial_debt_installments i
		JOIN financial_debts fd ON fd.id = i.financial_debt_id
		WHERE fd.status = 'active'
		AND i.status = 'overdue'
	`).Scan(&count).Error
	return count, err
}

func UpcomingFinancialDue(db *gorm.DB, start, end time.Time) (decimal.Decimal, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return decimal.Zero, err
	}
	return decimalFromQuery(db, `
		SELECT COALESCE(SUM(i.amount - i.paid_amount), 0)::text
		FROM financial_debt_installments i
		JOIN financial_debts fd ON fd.id = i.financial_debt_id
		WHERE fd.status = 'active'
		AND i.due_date >= ? AND i.due_date < ?
		AND i.status IN ('pending', 'partial_paid')
	`, start, end)
}

func FinancialDebtRows(db *gorm.DB, where string, args ...interface{}) ([]FinancialDebtRow, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return nil, err
	}
	query := `
		SELECT
			fd.id,
			fd.debt_type,
			fd.institution_name,
			fd.title,
			fd.total_amount::text AS total_amount,
			COALESCE(SUM(i.amount), 0)::text AS installment_total,
			COALESCE(SUM(i.paid_amount), 0)::text AS paid_total,
			COALESCE(SUM(i.amount - i.paid_amount), 0)::text AS remaining_amount,
			fd.start_date,
			fd.end_date,
			fd.status,
			fd.note,
			fd.created_at,
			fd.updated_at
		FROM financial_debts fd
		LEFT JOIN financial_debt_installments i ON i.financial_debt_id = fd.id
	`
	if where != "" {
		query += " WHERE " + where
	}
	query += `
		GROUP BY fd.id
		ORDER BY fd.end_date ASC, fd.id DESC
	`

	var rows []struct {
		ID               uint
		DebtType         string
		InstitutionName  string
		Title            string
		TotalAmount      string
		InstallmentTotal string
		PaidTotal        string
		RemainingAmount  string
		StartDate        time.Time
		EndDate          time.Time
		Status           string
		Note             string
		CreatedAt        time.Time
		UpdatedAt        time.Time
	}
	if err := db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]FinancialDebtRow, 0, len(rows))
	for _, row := range rows {
		totalAmount, err := decimal.NewFromString(row.TotalAmount)
		if err != nil {
			return nil, err
		}
		installmentTotal, err := decimal.NewFromString(row.InstallmentTotal)
		if err != nil {
			return nil, err
		}
		paidTotal, err := decimal.NewFromString(row.PaidTotal)
		if err != nil {
			return nil, err
		}
		remainingAmount, err := decimal.NewFromString(row.RemainingAmount)
		if err != nil {
			return nil, err
		}
		result = append(result, FinancialDebtRow{
			ID:               row.ID,
			DebtType:         row.DebtType,
			InstitutionName:  row.InstitutionName,
			Title:            row.Title,
			TotalAmount:      totalAmount,
			InstallmentTotal: installmentTotal,
			PaidTotal:        paidTotal,
			RemainingAmount:  remainingAmount,
			StartDate:        row.StartDate,
			EndDate:          row.EndDate,
			Status:           row.Status,
			Note:             row.Note,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return result, nil
}

func FinancialDebtRowByID(db *gorm.DB, debtID uint) (FinancialDebtRow, error) {
	rows, err := FinancialDebtRows(db, "fd.id = ?", debtID)
	if err != nil {
		return FinancialDebtRow{}, err
	}
	if len(rows) == 0 {
		return FinancialDebtRow{}, gorm.ErrRecordNotFound
	}
	return rows[0], nil
}

func FinancialDebtSummaryByID(db *gorm.DB, debtID uint) (FinancialDebtSummary, error) {
	row, err := FinancialDebtRowByID(db, debtID)
	if err != nil {
		return FinancialDebtSummary{}, err
	}
	var installments []models.FinancialDebtInstallment
	if err := db.Where("financial_debt_id = ?", debtID).Order("installment_no asc, due_date asc, id asc").Find(&installments).Error; err != nil {
		return FinancialDebtSummary{}, err
	}
	return FinancialDebtSummary{
		FinancialDebt:    row,
		InstallmentTotal: row.InstallmentTotal,
		PaidTotal:        row.PaidTotal,
		RemainingAmount:  row.RemainingAmount,
		Installments:     installments,
	}, nil
}

func NearestFinancialInstallments(db *gorm.DB, start, end time.Time, limit int) ([]models.FinancialDebtInstallment, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return nil, err
	}
	query := db.Joins("JOIN financial_debts fd ON fd.id = financial_debt_installments.financial_debt_id").
		Where("fd.status = 'active' AND financial_debt_installments.due_date >= ? AND financial_debt_installments.due_date < ? AND financial_debt_installments.status IN ?", start, end, []string{"pending", "partial_paid"}).
		Order("due_date asc, installment_no asc, id asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var installments []models.FinancialDebtInstallment
	if err := query.Find(&installments).Error; err != nil {
		return nil, err
	}
	return installments, nil
}

func FinancialAlertsData(db *gorm.DB) (FinancialAlerts, error) {
	if err := RefreshFinancialInstallmentStatuses(db); err != nil {
		return FinancialAlerts{}, err
	}
	today := startOfDay(time.Now())
	tomorrow := today.AddDate(0, 0, 1)

	alerts := FinancialAlerts{}
	activeInstallments := func() *gorm.DB {
		return db.Model(&models.FinancialDebtInstallment{}).
			Joins("JOIN financial_debts fd ON fd.id = financial_debt_installments.financial_debt_id").
			Where("fd.status = 'active'")
	}
	if err := activeInstallments().Where("financial_debt_installments.status = ?", "overdue").Order("due_date asc, financial_debt_installments.id asc").Find(&alerts.OverdueInstallments).Error; err != nil {
		return FinancialAlerts{}, err
	}
	if err := activeInstallments().Where("financial_debt_installments.due_date >= ? AND financial_debt_installments.due_date < ? AND financial_debt_installments.status IN ?", today, tomorrow, []string{"pending", "partial_paid"}).Order("financial_debt_installments.id asc").Find(&alerts.DueToday).Error; err != nil {
		return FinancialAlerts{}, err
	}
	if err := activeInstallments().Where("financial_debt_installments.due_date >= ? AND financial_debt_installments.due_date < ? AND financial_debt_installments.status IN ?", today, today.AddDate(0, 0, 8), []string{"pending", "partial_paid"}).Order("due_date asc, financial_debt_installments.id asc").Find(&alerts.DueIn7Days).Error; err != nil {
		return FinancialAlerts{}, err
	}
	if err := activeInstallments().Where("financial_debt_installments.due_date >= ? AND financial_debt_installments.due_date < ? AND financial_debt_installments.status IN ?", today, today.AddDate(0, 0, 31), []string{"pending", "partial_paid"}).Order("due_date asc, financial_debt_installments.id asc").Find(&alerts.DueIn30Days).Error; err != nil {
		return FinancialAlerts{}, err
	}
	return alerts, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
