package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type IncomeEntryHandler struct{ DB *gorm.DB }

type incomeEntryRequest struct {
	IncomeDate    string          `json:"income_date"`
	Category      string          `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	Note          string          `json:"note"`
}

func NewIncomeEntryHandler(db *gorm.DB) *IncomeEntryHandler {
	return &IncomeEntryHandler{DB: db}
}

func (h *IncomeEntryHandler) Create(c *gin.Context) {
	var req incomeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	entry, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&entry).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, entry)
}

func (h *IncomeEntryHandler) List(c *gin.Context) {
	var entries []models.IncomeEntry
	query := h.DB.Order("income_date desc, id desc")
	if startDate := c.Query("start_date"); startDate != "" {
		date, err := parseDate(startDate)
		if err != nil {
			fail(c, http.StatusBadRequest, "start_date is invalid")
			return
		}
		query = query.Where("income_date >= ?", date)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		date, err := parseDate(endDate)
		if err != nil {
			fail(c, http.StatusBadRequest, "end_date is invalid")
			return
		}
		query = query.Where("income_date <= ?", date)
	}
	if err := query.Find(&entries).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, entries)
}

func (h *IncomeEntryHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req incomeEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	entry, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.IncomeEntry
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	entry.ID = existing.ID
	entry.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&entry).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, entry)
}

func (h *IncomeEntryHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.IncomeEntry{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r incomeEntryRequest) toModel() (models.IncomeEntry, error) {
	if strings.TrimSpace(r.Category) == "" {
		return models.IncomeEntry{}, errRequired("category")
	}
	if !validateType(r.Category, map[string]bool{
		"market_satis":      true,
		"tup_satis":         true,
		"veresiye_tahsilat": true,
		"diger":             true,
	}) {
		return models.IncomeEntry{}, errInvalidType("category")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.IncomeEntry{}, err
	}
	date, err := parseDate(r.IncomeDate)
	if err != nil {
		return models.IncomeEntry{}, err
	}
	return models.IncomeEntry{
		IncomeDate:    date,
		Category:      r.Category,
		Amount:        r.Amount,
		PaymentMethod: strings.TrimSpace(r.PaymentMethod),
		Note:          strings.TrimSpace(r.Note),
	}, nil
}
