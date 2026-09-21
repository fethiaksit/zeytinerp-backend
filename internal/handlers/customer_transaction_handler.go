package handlers

import (
	"net/http"
	"strings"
	"time"

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

type customerTransactionResponse struct {
	ID              uint            `json:"id"`
	CustomerID      uint            `json:"customer_id"`
	TransactionDate time.Time       `json:"transaction_date"`
	Type            string          `json:"type"`
	Amount          decimal.Decimal `json:"amount"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	Note            string          `json:"note"`
	CreatedAt       time.Time       `json:"created_at"`
}

func NewCustomerTransactionHandler(db *gorm.DB) *CustomerTransactionHandler {
	return &CustomerTransactionHandler{DB: db}
}

// POST /api/customer-transactions or /api/customers/:id/transactions
func (h *CustomerTransactionHandler) Create(c *gin.Context) {
	var req customerTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	if customerIDStr := c.Param("id"); customerIDStr != "" {
		id, valid := parseID(c)
		if !valid {
			return
		}
		req.CustomerID = id
	}

	var customer models.Customer
	if err := h.DB.First(&customer, req.CustomerID).Error; err != nil {
		handleDBError(c, err)
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

	// Compute running balance after creation
	var allTxs []models.CustomerTransaction
	if err := h.DB.Where("customer_id = ?", req.CustomerID).Order("transaction_date asc, id asc").Find(&allTxs).Error; err != nil {
		handleDBError(c, err)
		return
	}

	runningBalance := decimal.Zero
	var currentBalanceAfter decimal.Decimal
	for _, item := range allTxs {
		if strings.EqualFold(item.Type, "debt") || strings.EqualFold(item.Type, "sale") {
			runningBalance = runningBalance.Add(item.Amount)
		} else if strings.EqualFold(item.Type, "payment") || strings.EqualFold(item.Type, "return") {
			runningBalance = runningBalance.Sub(item.Amount)
		}
		if item.ID == tx.ID {
			currentBalanceAfter = runningBalance
		}
	}

	res := customerTransactionResponse{
		ID:              tx.ID,
		CustomerID:      tx.CustomerID,
		TransactionDate: tx.TransactionDate,
		Type:            tx.Type,
		Amount:          tx.Amount,
		BalanceAfter:    currentBalanceAfter,
		Note:            tx.Note,
		CreatedAt:       tx.CreatedAt,
	}

	created(c, res)
}

// GET /api/customers/:id/transactions or /api/customer-transactions
func (h *CustomerTransactionHandler) List(c *gin.Context) {
	var customerID uint
	if customerIDStr := c.Param("id"); customerIDStr != "" {
		id, valid := parseID(c)
		if !valid {
			return
		}
		customerID = id
	}

	if customerID == 0 {
		if qCustID := c.Query("customer_id"); qCustID != "" {
			var cust models.Customer
			if err := h.DB.First(&cust, qCustID).Error; err == nil {
				customerID = cust.ID
			}
		}
	}

	if customerID == 0 {
		var txs []models.CustomerTransaction
		if err := h.DB.Order("transaction_date desc, id desc").Limit(100).Find(&txs).Error; err != nil {
			handleDBError(c, err)
			return
		}
		ok(c, txs)
		return
	}

	// Fetch all transactions for customer ordered chronologically to compute running balance
	var allTxs []models.CustomerTransaction
	if err := h.DB.Where("customer_id = ?", customerID).Order("transaction_date asc, id asc").Find(&allTxs).Error; err != nil {
		handleDBError(c, err)
		return
	}

	runningBalance := decimal.Zero
	responsesWithBalance := make([]customerTransactionResponse, 0, len(allTxs))

	for _, tx := range allTxs {
		if strings.EqualFold(tx.Type, "debt") || strings.EqualFold(tx.Type, "sale") {
			runningBalance = runningBalance.Add(tx.Amount)
		} else if strings.EqualFold(tx.Type, "payment") || strings.EqualFold(tx.Type, "return") {
			runningBalance = runningBalance.Sub(tx.Amount)
		}

		responsesWithBalance = append(responsesWithBalance, customerTransactionResponse{
			ID:              tx.ID,
			CustomerID:      tx.CustomerID,
			TransactionDate: tx.TransactionDate,
			Type:            tx.Type,
			Amount:          tx.Amount,
			BalanceAfter:    runningBalance,
			Note:            tx.Note,
			CreatedAt:       tx.CreatedAt,
		})
	}

	// Read filters
	startDateStr := strings.TrimSpace(c.Query("start_date"))
	if startDateStr == "" {
		startDateStr = strings.TrimSpace(c.Query("startDate"))
	}

	endDateStr := strings.TrimSpace(c.Query("end_date"))
	if endDateStr == "" {
		endDateStr = strings.TrimSpace(c.Query("endDate"))
	}

	typeFilter := strings.TrimSpace(strings.ToLower(c.Query("type")))

	filtered := make([]customerTransactionResponse, 0, len(responsesWithBalance))
	for _, item := range responsesWithBalance {
		dateStr := item.TransactionDate.Format("2006-01-02")
		if startDateStr != "" && dateStr < startDateStr {
			continue
		}
		if endDateStr != "" && dateStr > endDateStr {
			continue
		}
		if typeFilter != "" && typeFilter != "all" {
			itemTypeLower := strings.ToLower(item.Type)
			if typeFilter == "debt" && itemTypeLower != "debt" && itemTypeLower != "sale" {
				continue
			} else if typeFilter == "payment" && itemTypeLower != "payment" {
				continue
			} else if typeFilter == "sale" && itemTypeLower != "sale" {
				continue
			} else if typeFilter == "return" && itemTypeLower != "return" {
				continue
			} else if typeFilter != "debt" && typeFilter != "payment" && typeFilter != "sale" && typeFilter != "return" && itemTypeLower != typeFilter {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	// Reverse to return descending order (newest first)
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	ok(c, filtered)
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
	var date time.Time
	var err error
	if r.TransactionDate != "" {
		date, err = parseDate(r.TransactionDate)
		if err != nil {
			return models.CustomerTransaction{}, err
		}
	} else {
		date = time.Now()
	}
	return models.CustomerTransaction{
		CustomerID:      r.CustomerID,
		TransactionDate: date,
		Type:            r.Type,
		Amount:          r.Amount,
		Note:            r.Note,
	}, nil
}
