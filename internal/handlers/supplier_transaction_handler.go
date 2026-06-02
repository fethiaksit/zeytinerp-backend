package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type SupplierTransactionHandler struct{ DB *gorm.DB }

type supplierTransactionRequest struct {
	SupplierID      uint            `json:"supplier_id"`
	TransactionDate string          `json:"transaction_date"`
	Type            string          `json:"type"`
	Amount          decimal.Decimal `json:"amount"`
	PaymentMethod   string          `json:"payment_method"`
	Note            string          `json:"note"`
}

func NewSupplierTransactionHandler(db *gorm.DB) *SupplierTransactionHandler {
	return &SupplierTransactionHandler{DB: db}
}

func (h *SupplierTransactionHandler) Create(c *gin.Context) {
	var req supplierTransactionRequest
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

func (h *SupplierTransactionHandler) List(c *gin.Context) {
	var txs []models.SupplierTransaction
	query := h.DB.Order("transaction_date desc, id desc")
	if supplierID := c.Query("supplier_id"); supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if err := query.Find(&txs).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, txs)
}

func (h *SupplierTransactionHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.SupplierTransaction{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r supplierTransactionRequest) toModel() (models.SupplierTransaction, error) {
	if r.SupplierID == 0 {
		return models.SupplierTransaction{}, errRequired("supplier_id")
	}
	if !validateType(r.Type, map[string]bool{"purchase": true, "payment": true}) {
		return models.SupplierTransaction{}, errInvalidType("type")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.SupplierTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.SupplierTransaction{}, err
	}
	return models.SupplierTransaction{
		SupplierID:      r.SupplierID,
		TransactionDate: date,
		Type:            r.Type,
		Amount:          r.Amount,
		PaymentMethod:   r.PaymentMethod,
		Note:            r.Note,
	}, nil
}
