package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type EmployeeTransactionHandler struct{ DB *gorm.DB }

type employeeTransactionRequest struct {
	EmployeeID      uint            `json:"employee_id"`
	TransactionDate string          `json:"transaction_date"`
	Type            string          `json:"type"`
	WorkDays        decimal.Decimal `json:"work_days"`
	Amount          decimal.Decimal `json:"amount"`
	Note            string          `json:"note"`
}

func NewEmployeeTransactionHandler(db *gorm.DB) *EmployeeTransactionHandler {
	return &EmployeeTransactionHandler{DB: db}
}

func (h *EmployeeTransactionHandler) Create(c *gin.Context) {
	var req employeeTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	tx, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&tx).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, tx)
}

func (h *EmployeeTransactionHandler) List(c *gin.Context) {
	var txs []models.EmployeeTransaction
	query := h.DB.Order("transaction_date desc, id desc")
	if employeeID := c.Query("employee_id"); employeeID != "" {
		query = query.Where("employee_id = ?", employeeID)
	}
	if err := query.Find(&txs).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, txs)
}

func (h *EmployeeTransactionHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.EmployeeTransaction{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r employeeTransactionRequest) toModel() (models.EmployeeTransaction, error) {
	if r.EmployeeID == 0 {
		return models.EmployeeTransaction{}, errRequired("employee_id")
	}
	if !validateType(r.Type, map[string]bool{"work": true, "payment": true, "advance": true}) {
		return models.EmployeeTransaction{}, errInvalidType("type")
	}
	if r.Type == "work" {
		if err := positiveDecimal(r.WorkDays, "work_days"); err != nil {
			return models.EmployeeTransaction{}, err
		}
	} else if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.EmployeeTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.EmployeeTransaction{}, err
	}
	return models.EmployeeTransaction{EmployeeID: r.EmployeeID, TransactionDate: date, Type: r.Type, WorkDays: r.WorkDays, Amount: r.Amount, Note: r.Note}, nil
}
