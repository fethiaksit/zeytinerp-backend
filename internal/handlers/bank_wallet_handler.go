package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type BankWalletHandler struct{ DB *gorm.DB }

type bankAccountRequest struct {
	AccountName    string          `json:"account_name"`
	BankName       string          `json:"bank_name"`
	IBAN           string          `json:"iban"`
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	IsActive       *bool           `json:"is_active"`
}

type bankTransactionRequest struct {
	TransactionDate string          `json:"transaction_date"`
	TransactionType string          `json:"transaction_type"`
	Amount          decimal.Decimal `json:"amount"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	RelatedType     string          `json:"related_type"`
	RelatedID       *uint           `json:"related_id"`
}

func NewBankWalletHandler(db *gorm.DB) *BankWalletHandler {
	return &BankWalletHandler{DB: db}
}

func (h *BankWalletHandler) ListAccounts(c *gin.Context) {
	var accounts []models.BankAccount
	if err := h.DB.Where("is_active = true").Order("id desc").Find(&accounts).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, accounts)
}

func (h *BankWalletHandler) CreateAccount(c *gin.Context) {
	var req bankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	account, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
		openingTransaction := models.BankTransaction{
			BankAccountID:   account.ID,
			TransactionDate: time.Now(),
			TransactionType: "opening_balance",
			Amount:          account.OpeningBalance,
			BalanceAfter:    account.OpeningBalance,
			Title:           "Açılış Bakiyesi",
			Description:     "Hesap açılış bakiyesi",
		}
		if err := tx.Create(&openingTransaction).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, account)
}

func (h *BankWalletHandler) GetAccount(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var account models.BankAccount
	if err := h.DB.Preload("Transactions", func(db *gorm.DB) *gorm.DB {
		return db.Order("transaction_date desc, id desc")
	}).First(&account, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, account)
}

func (h *BankWalletHandler) UpdateAccount(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var existing models.BankAccount
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	var req bankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	updated, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updates := map[string]interface{}{
		"account_name": updated.AccountName,
		"bank_name":    updated.BankName,
		"iban":         updated.IBAN,
		"is_active":    updated.IsActive,
	}
	if err := h.DB.Model(&models.BankAccount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, existing)
}

func (h *BankWalletHandler) DeleteAccount(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Model(&models.BankAccount{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *BankWalletHandler) ListTransactions(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var account models.BankAccount
	if err := h.DB.First(&account, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	var transactions []models.BankTransaction
	if err := h.DB.Where("bank_account_id = ?", id).
		Order("transaction_date desc, id desc").
		Find(&transactions).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, transactions)
}

func (h *BankWalletHandler) CreateTransaction(c *gin.Context) {
	accountID, valid := parseID(c)
	if !valid {
		return
	}
	var req bankTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	transaction, err := req.toModel(accountID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	transaction, err = services.CreateBankTransaction(h.DB, transaction)
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, transaction)
}

func (h *BankWalletHandler) DeleteTransaction(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var transaction models.BankTransaction
	if err := h.DB.First(&transaction, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.BankTransaction{}, id).Error; err != nil {
			return err
		}
		return services.RecalculateBankAccountBalance(tx, transaction.BankAccountID)
	})
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *BankWalletHandler) Summary(c *gin.Context) {
	today := todayStart()
	summary, err := services.BankWalletOverview(h.DB, today)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, summary)
}

func (h *BankWalletHandler) DailySummary(c *gin.Context) {
	dateValue := c.DefaultQuery("date", todayStart().Format("2006-01-02"))
	date, err := parseDate(dateValue)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	income, outcome, transactions, err := services.BankPeriodSummaryData(h.DB, start, start.AddDate(0, 0, 1))
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, services.BankPeriodSummary{
		Date:         start.Format("2006-01-02"),
		Income:       income,
		Outcome:      outcome,
		Net:          income.Sub(outcome),
		Transactions: transactions,
	})
}

func (h *BankWalletHandler) MonthlySummary(c *gin.Context) {
	monthValue := c.DefaultQuery("month", time.Now().Format("2006-01"))
	month, err := time.Parse("2006-01", monthValue)
	if err != nil {
		fail(c, http.StatusBadRequest, "month is invalid")
		return
	}
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)
	income, outcome, transactions, err := services.BankPeriodSummaryData(h.DB, start, start.AddDate(0, 1, 0))
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, services.BankPeriodSummary{
		Month:        start.Format("2006-01"),
		Income:       income,
		Outcome:      outcome,
		Net:          income.Sub(outcome),
		Transactions: transactions,
	})
}

func (r bankAccountRequest) toModel() (models.BankAccount, error) {
	if strings.TrimSpace(r.AccountName) == "" {
		return models.BankAccount{}, errRequired("account_name")
	}
	if strings.TrimSpace(r.BankName) == "" {
		return models.BankAccount{}, errRequired("bank_name")
	}
	if r.OpeningBalance.IsNegative() {
		return models.BankAccount{}, errInvalidType("opening_balance")
	}
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return models.BankAccount{
		AccountName:    strings.TrimSpace(r.AccountName),
		BankName:       strings.TrimSpace(r.BankName),
		IBAN:           strings.TrimSpace(r.IBAN),
		OpeningBalance: r.OpeningBalance,
		CurrentBalance: r.OpeningBalance,
		IsActive:       active,
	}, nil
}

func (r bankTransactionRequest) toModel(accountID uint) (models.BankTransaction, error) {
	if accountID == 0 {
		return models.BankTransaction{}, errRequired("bank_account_id")
	}
	r.TransactionType = strings.TrimSpace(r.TransactionType)
	if !validateType(r.TransactionType, map[string]bool{
		"opening_balance": true,
		"cash_deposit":    true,
		"pos_income":      true,
		"bank_income":     true,
		"payment":         true,
		"expense":         true,
		"transfer_in":     true,
		"transfer_out":    true,
		"correction":      true,
	}) {
		return models.BankTransaction{}, errInvalidType("transaction_type")
	}
	if r.TransactionType == "correction" {
		if r.Amount.Equal(decimal.Zero) {
			return models.BankTransaction{}, errRequired("amount")
		}
	} else if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.BankTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.BankTransaction{}, err
	}
	return models.BankTransaction{
		BankAccountID:   accountID,
		TransactionDate: date,
		TransactionType: r.TransactionType,
		Amount:          r.Amount,
		Title:           strings.TrimSpace(r.Title),
		Description:     strings.TrimSpace(r.Description),
		RelatedType:     strings.TrimSpace(r.RelatedType),
		RelatedID:       r.RelatedID,
	}, nil
}

func todayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}
