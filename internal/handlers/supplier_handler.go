package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type SupplierHandler struct{ DB *gorm.DB }

type supplierRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	Note     string `json:"note"`
	IsActive *bool  `json:"is_active"`
}

func NewSupplierHandler(db *gorm.DB) *SupplierHandler { return &SupplierHandler{DB: db} }

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
	ok(c, gin.H{"supplier_id": id, "balance": balance})
}

func (h *SupplierHandler) Balances(c *gin.Context) {
	balances, err := services.SupplierBalances(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, balances)
}

func (r supplierRequest) toModel() (models.Supplier, error) {
	if strings.TrimSpace(r.Name) == "" {
		return models.Supplier{}, errRequired("name")
	}
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return models.Supplier{
		Name:     strings.TrimSpace(r.Name),
		Phone:    strings.TrimSpace(r.Phone),
		Address:  strings.TrimSpace(r.Address),
		Note:     strings.TrimSpace(r.Note),
		IsActive: active,
	}, nil
}
