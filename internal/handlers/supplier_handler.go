package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type SupplierHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

type supplierRequest struct {
	Name      string   `json:"name"`
	Phone     string   `json:"phone"`
	Address   string   `json:"address"`
	Note      string   `json:"note"`
	VisitDays []string `json:"visit_days"`
	IsActive  *bool    `json:"is_active"`
}

var visitDayOrder = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
}

var visitDayAliases = map[string]string{
	"monday": "monday", "pazartesi": "monday",
	"tuesday": "tuesday", "salı": "tuesday", "sali": "tuesday",
	"wednesday": "wednesday", "çarşamba": "wednesday", "carsamba": "wednesday",
	"thursday": "thursday", "perşembe": "thursday", "persembe": "thursday",
	"friday": "friday", "cuma": "friday",
	"saturday": "saturday", "cumartesi": "saturday",
	"sunday": "sunday", "pazar": "sunday",
}

func NewSupplierHandler(db *gorm.DB) *SupplierHandler {
	return &SupplierHandler{DB: db, Now: time.Now}
}

func (h *SupplierHandler) Create(c *gin.Context) {
	var req supplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	supplier, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&supplier).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, supplier)
}

func (h *SupplierHandler) List(c *gin.Context) {
	var suppliers []models.Supplier
	query := h.DB.Order("id desc")
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR phone ILIKE ? OR note ILIKE ?", like, like, like)
	}
	if err := query.Find(&suppliers).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, suppliers)
}

func (h *SupplierHandler) TodayVisits(c *gin.Context) {
	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}
	location, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		location = time.FixedZone("Europe/Istanbul", 3*60*60)
	}
	h.listVisitsByDay(c, strings.ToLower(now.In(location).Weekday().String()))
}

func (h *SupplierHandler) Visits(c *gin.Context) {
	day := strings.TrimSpace(c.Query("day"))
	if day == "" {
		fail(c, http.StatusBadRequest, "day is required")
		return
	}
	normalized, valid := normalizeVisitDay(day)
	if !valid {
		fail(c, http.StatusBadRequest, "day is invalid")
		return
	}
	h.listVisitsByDay(c, normalized)
}

func (h *SupplierHandler) listVisitsByDay(c *gin.Context, day string) {
	var suppliers []models.Supplier
	query := h.DB.Where("is_active = ?", true).Order("id desc")
	if h.DB.Dialector.Name() == "sqlite" {
		query = query.Where("EXISTS (SELECT 1 FROM json_each(suppliers.visit_days) WHERE json_each.value = ?)", day)
	} else {
		encoded, _ := json.Marshal([]string{day})
		query = query.Where("visit_days @> ?::jsonb", string(encoded))
	}
	if err := query.Find(&suppliers).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, suppliers)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var supplier models.Supplier
	if err := h.DB.First(&supplier, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, supplier)
}

func (h *SupplierHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var supplier models.Supplier
	if err := h.DB.First(&supplier, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	var req supplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	updated, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	updated.ID = id
	updated.CreatedAt = supplier.CreatedAt
	if err := h.DB.Save(&updated).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, updated)
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Supplier{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *SupplierHandler) Balance(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var supplier models.Supplier
	if err := h.DB.First(&supplier, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	balance, err := services.SupplierBalance(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	currencyTotals, err := services.SupplierCurrencyBalanceTotals(h.DB, &id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"supplier_id": id, "balance": balance, "currency_totals": currencyTotals})
}

func (h *SupplierHandler) Balances(c *gin.Context) {
	balances, err := services.SupplierBalances(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, balances)
}

func (h *SupplierHandler) CurrencyTotals(c *gin.Context) {
	totals, err := services.SupplierCurrencyBalanceTotals(h.DB, nil)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, totals)
}

func (r supplierRequest) toModel() (models.Supplier, error) {
	if strings.TrimSpace(r.Name) == "" {
		return models.Supplier{}, errRequired("name")
	}
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	visitDays, err := normalizeVisitDays(r.VisitDays)
	if err != nil {
		return models.Supplier{}, err
	}
	return models.Supplier{
		Name:      strings.TrimSpace(r.Name),
		Phone:     strings.TrimSpace(r.Phone),
		Address:   strings.TrimSpace(r.Address),
		Note:      strings.TrimSpace(r.Note),
		VisitDays: visitDays,
		IsActive:  active,
	}, nil
}

func normalizeVisitDays(days []string) ([]string, error) {
	selected := make(map[string]bool, len(days))
	for _, day := range days {
		normalized, valid := normalizeVisitDay(day)
		if !valid {
			return nil, errInvalidType("visit_days")
		}
		selected[normalized] = true
	}

	normalized := make([]string, 0, len(selected))
	for _, day := range visitDayOrder {
		if selected[day] {
			normalized = append(normalized, day)
		}
	}
	return normalized, nil
}

func normalizeVisitDay(day string) (string, bool) {
	normalized, valid := visitDayAliases[strings.ToLower(strings.TrimSpace(day))]
	return normalized, valid
}
