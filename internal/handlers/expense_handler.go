package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type ExpenseHandler struct{ DB *gorm.DB }

type expenseRequest struct {
	ExpenseDate   string          `json:"expense_date"`
	Category      string          `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	Note          string          `json:"note"`
}

func NewExpenseHandler(db *gorm.DB) *ExpenseHandler { return &ExpenseHandler{DB: db} }

func (h *ExpenseHandler) Create(c *gin.Context) {
	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	expense, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&expense).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, expense)
}

func (h *ExpenseHandler) List(c *gin.Context) {
	var expenses []models.Expense
	query := h.DB.Order("expense_date desc, id desc")
	if startDate := c.Query("start_date"); startDate != "" {
		date, err := parseDate(startDate)
		if err != nil {
			fail(c, http.StatusBadRequest, "start_date is invalid")
			return
		}
		query = query.Where("expense_date >= ?", date)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		date, err := parseDate(endDate)
		if err != nil {
			fail(c, http.StatusBadRequest, "end_date is invalid")
			return
		}
		query = query.Where("expense_date <= ?", date)
	}
	if err := query.Find(&expenses).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, expenses)
}

func (h *ExpenseHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	expense, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.Expense
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	expense.ID = existing.ID
	expense.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&expense).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, expense)
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Expense{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r expenseRequest) toModel() (models.Expense, error) {
	if strings.TrimSpace(r.Category) == "" {
		return models.Expense{}, errRequired("category")
	}
	if !validateType(r.Category, map[string]bool{
		"kira":          true,
		"elektrik":      true,
		"su":            true,
		"personel":      true,
		"yakit":         true,
		"yemek":         true,
		"market_gideri": true,
		"diger":         true,
	}) {
		return models.Expense{}, errInvalidType("category")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.Expense{}, err
	}
	date, err := parseDate(r.ExpenseDate)
	if err != nil {
		return models.Expense{}, err
	}
	return models.Expense{
		ExpenseDate:   date,
		Category:      r.Category,
		Amount:        r.Amount,
		PaymentMethod: strings.TrimSpace(r.PaymentMethod),
		Note:          strings.TrimSpace(r.Note),
	}, nil
}
