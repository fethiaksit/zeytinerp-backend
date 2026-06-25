package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type FinancialDebtHandler struct{ DB *gorm.DB }

type financialDebtRequest struct {
	DebtType        string          `json:"debt_type"`
	InstitutionName string          `json:"institution_name"`
	Title           string          `json:"title"`
	TotalAmount     decimal.Decimal `json:"total_amount"`
	StartDate       string          `json:"start_date"`
	EndDate         string          `json:"end_date"`
	Status          string          `json:"status"`
	Note            string          `json:"note"`
}

type financialInstallmentRequest struct {
	InstallmentNo int             `json:"installment_no"`
	DueDate       string          `json:"due_date"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note"`
}

type financialInstallmentBulkRequest struct {
	Installments []financialInstallmentRequest `json:"installments"`
}

func NewFinancialDebtHandler(db *gorm.DB) *FinancialDebtHandler {
	return &FinancialDebtHandler{DB: db}
}

func (h *FinancialDebtHandler) Create(c *gin.Context) {
	var req financialDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	debt, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&debt).Error; err != nil {
		handleDBError(c, err)
		return
	}
	row, err := services.FinancialDebtRowByID(h.DB, debt.ID)
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, row)
}

func (h *FinancialDebtHandler) List(c *gin.Context) {
	rows, err := services.FinancialDebtRows(h.DB, "")
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, rows)
}

func (h *FinancialDebtHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	row, err := services.FinancialDebtRowByID(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, row)
}

func (h *FinancialDebtHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var existing models.FinancialDebt
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	var req financialDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	debt, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	debt.ID = existing.ID
	debt.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&debt).Error; err != nil {
		handleDBError(c, err)
		return
	}
	row, err := services.FinancialDebtRowByID(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, row)
}

func (h *FinancialDebtHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.FinancialDebt{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *FinancialDebtHandler) Balance(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	balance, err := services.FinancialDebtBalance(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"financial_debt_id": id, "balance": balance})
}

func (h *FinancialDebtHandler) Summary(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	summary, err := services.FinancialDebtSummaryByID(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, summary)
}

func (h *FinancialDebtHandler) CreateInstallment(c *gin.Context) {
	debtID, valid := parseID(c)
	if !valid {
		return
	}
	var req financialInstallmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	installment, err := req.toModel(debtID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.ensureDebtExists(debtID); err != nil {
		handleDBError(c, err)
		return
	}
	if err := h.DB.Create(&installment).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if err := services.RecalculateFinancialInstallment(h.DB, installment.ID); err != nil {
		handleDBError(c, err)
		return
	}
	if err := h.DB.First(&installment, installment.ID).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, installment)
}

func (h *FinancialDebtHandler) ListInstallments(c *gin.Context) {
	debtID, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.ensureDebtExists(debtID); err != nil {
		handleDBError(c, err)
		return
	}
	if err := services.RefreshFinancialInstallmentStatuses(h.DB); err != nil {
		handleDBError(c, err)
		return
	}
	var installments []models.FinancialDebtInstallment
	if err := h.DB.Where("financial_debt_id = ?", debtID).Order("installment_no asc, due_date asc, id asc").Find(&installments).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, installments)
}

func (h *FinancialDebtHandler) BulkCreateInstallments(c *gin.Context) {
	debtID, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.ensureDebtExists(debtID); err != nil {
		handleDBError(c, err)
		return
	}

	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	requests, err := parseInstallmentBulk(raw)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	installments := make([]models.FinancialDebtInstallment, 0, len(requests))
	for _, req := range requests {
		installment, err := req.toModel(debtID)
		if err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		installments = append(installments, installment)
	}
	if len(installments) == 0 {
		fail(c, http.StatusBadRequest, "installments is required")
		return
	}
	if err := h.DB.Create(&installments).Error; err != nil {
		handleDBError(c, err)
		return
	}
	for _, installment := range installments {
		if err := services.RecalculateFinancialInstallment(h.DB, installment.ID); err != nil {
			handleDBError(c, err)
			return
		}
	}
	ok(c, installments)
}

func (h *FinancialDebtHandler) UpdateInstallment(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var existing models.FinancialDebtInstallment
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	var req financialInstallmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	installment, err := req.toModel(existing.FinancialDebtID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	installment.ID = existing.ID
	installment.CreatedAt = existing.CreatedAt
	installment.PaidAmount = existing.PaidAmount
	installment.Status = existing.Status
	if err := h.DB.Save(&installment).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if err := services.RecalculateFinancialInstallment(h.DB, id); err != nil {
		handleDBError(c, err)
		return
	}
	if err := h.DB.First(&installment, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, installment)
}

func (h *FinancialDebtHandler) DeleteInstallment(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.FinancialDebtInstallment{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *FinancialDebtHandler) Alerts(c *gin.Context) {
	alerts, err := services.FinancialAlertsData(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, alerts)
}

func (h *FinancialDebtHandler) ensureDebtExists(id uint) error {
	var debt models.FinancialDebt
	return h.DB.First(&debt, id).Error
}

func (r financialDebtRequest) toModel() (models.FinancialDebt, error) {
	r.DebtType = strings.TrimSpace(r.DebtType)
	r.InstitutionName = strings.TrimSpace(r.InstitutionName)
	r.Title = strings.TrimSpace(r.Title)
	r.Status = strings.TrimSpace(r.Status)
	if r.Status == "" {
		r.Status = "active"
	}
	if !validateType(r.DebtType, map[string]bool{
		"bank_loan":        true,
		"credit_card":      true,
		"installment_debt": true,
		"other":            true,
	}) {
		return models.FinancialDebt{}, errInvalidType("debt_type")
	}
	if !validateType(r.Status, map[string]bool{"active": true, "closed": true}) {
		return models.FinancialDebt{}, errInvalidType("status")
	}
	if r.InstitutionName == "" {
		return models.FinancialDebt{}, errRequired("institution_name")
	}
	if r.Title == "" {
		return models.FinancialDebt{}, errRequired("title")
	}
	if err := positiveDecimal(r.TotalAmount, "total_amount"); err != nil {
		return models.FinancialDebt{}, err
	}
	startDate, err := parseDate(r.StartDate)
	if err != nil {
		return models.FinancialDebt{}, err
	}
	endDate, err := parseDate(r.EndDate)
	if err != nil {
		return models.FinancialDebt{}, err
	}
	return models.FinancialDebt{
		DebtType:        r.DebtType,
		InstitutionName: r.InstitutionName,
		Title:           r.Title,
		TotalAmount:     r.TotalAmount,
		StartDate:       startDate,
		EndDate:         endDate,
		DueDate:         endDate,
		Status:          r.Status,
		Note:            strings.TrimSpace(r.Note),
	}, nil
}

func (r financialInstallmentRequest) toModel(debtID uint) (models.FinancialDebtInstallment, error) {
	if r.InstallmentNo <= 0 {
		return models.FinancialDebtInstallment{}, errInvalidType("installment_no")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.FinancialDebtInstallment{}, err
	}
	dueDate, err := parseDate(r.DueDate)
	if err != nil {
		return models.FinancialDebtInstallment{}, err
	}
	return models.FinancialDebtInstallment{
		FinancialDebtID: debtID,
		InstallmentNo:   r.InstallmentNo,
		DueDate:         dueDate,
		Amount:          r.Amount,
		Note:            strings.TrimSpace(r.Note),
	}, nil
}

func parseInstallmentBulk(raw json.RawMessage) ([]financialInstallmentRequest, error) {
	var wrapped financialInstallmentBulkRequest
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Installments != nil {
		return wrapped.Installments, nil
	}
	var direct []financialInstallmentRequest
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, errInvalidType("installments")
	}
	return direct, nil
}
