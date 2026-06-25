package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type CustomerTransactionHandler struct{ DB *gorm.DB }

type customerTransactionRequest struct {
	CustomerID      uint            `json:"customer_id"`
	TransactionDate string          `json:"transaction_date"`
	Type            string          `json:"type"`
	Amount          decimal.Decimal `json:"amount"`
	Note            string          `json:"note"`
}

func NewCustomerTransactionHandler(db *gorm.DB) *CustomerTransactionHandler {
	return &CustomerTransactionHandler{DB: db}
}

func (h *CustomerTransactionHandler) Create(c *gin.Context) {
	var req customerTransactionRequest
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

func (h *CustomerTransactionHandler) List(c *gin.Context) {
	var txs []models.CustomerTransaction
	query := h.DB.Order("transaction_date desc, id desc")
	if customerID := c.Query("customer_id"); customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if err := query.Find(&txs).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, txs)
}

func (h *CustomerTransactionHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.CustomerTransaction{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r customerTransactionRequest) toModel() (models.CustomerTransaction, error) {
	if r.CustomerID == 0 {
		return models.CustomerTransaction{}, errRequired("customer_id")
	}
	if !validateType(r.Type, map[string]bool{"debt": true, "payment": true}) {
		return models.CustomerTransaction{}, errInvalidType("type")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.CustomerTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.CustomerTransaction{}, err
	}
	return models.CustomerTransaction{CustomerID: r.CustomerID, TransactionDate: date, Type: r.Type, Amount: r.Amount, Note: r.Note}, nil
}
