package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type WalletHandler struct{ DB *gorm.DB }

type walletTransactionRequest struct {
	TransactionDate string          `json:"transaction_date"`
	TransactionType string          `json:"transaction_type"`
	Amount          decimal.Decimal `json:"amount"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	RelatedType     string          `json:"related_type"`
	RelatedID       *uint           `json:"related_id"`
}

func NewWalletHandler(db *gorm.DB) *WalletHandler {
	return &WalletHandler{DB: db}
}

func (h *WalletHandler) Summary(c *gin.Context) {
	today := todayStart()
	summary, err := services.WalletOverview(h.DB, today)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, summary)
}

func (h *WalletHandler) ListTransactions(c *gin.Context) {
	var transactions []models.WalletTransaction
	if err := h.DB.Order("transaction_date desc, id desc").Find(&transactions).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, transactions)
}

func (h *WalletHandler) CreateTransaction(c *gin.Context) {
	var req walletTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	transaction, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	transaction, err = services.CreateWalletTransaction(h.DB, transaction)
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, transaction)
}

func (h *WalletHandler) DeleteTransaction(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var transaction models.WalletTransaction
	if err := h.DB.First(&transaction, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if err := services.DeleteWalletTransaction(h.DB, id); err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r walletTransactionRequest) toModel() (models.WalletTransaction, error) {
	r.TransactionType = strings.TrimSpace(r.TransactionType)
	if !validateType(r.TransactionType, map[string]bool{
		"opening_balance": true,
		"cash_income":     true,
		"cash_sale":       true,
		"pos_income":      true,
		"bank_income":     true,
		"payment":         true,
		"expense":         true,
		"cash_withdraw":   true,
		"cash_deposit":    true,
		"correction":      true,
	}) {
		return models.WalletTransaction{}, errInvalidType("transaction_type")
	}
	if r.TransactionType == "correction" {
		if r.Amount.Equal(decimal.Zero) {
			return models.WalletTransaction{}, errRequired("amount")
		}
	} else if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.WalletTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.WalletTransaction{}, err
	}
	return models.WalletTransaction{
		TransactionDate: date,
		TransactionType: r.TransactionType,
		Amount:          r.Amount,
		Title:           strings.TrimSpace(r.Title),
		Description:     strings.TrimSpace(r.Description),
		RelatedType:     strings.TrimSpace(r.RelatedType),
		RelatedID:       r.RelatedID,
	}, nil
}
