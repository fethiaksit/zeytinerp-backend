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

type CustomerHandler struct {
	DB *gorm.DB
}

func NewCustomerHandler(db *gorm.DB) *CustomerHandler {
	return &CustomerHandler{DB: db}
}

type CustomerResponse struct {
	ID           uint             `json:"id"`
	Name         string           `json:"name"`
	Phone        string           `json:"phone"`
	Address      string           `json:"address"`
	CustomerType string           `json:"customer_type"`
	Note         string           `json:"note"`
	IsActive     bool             `json:"is_active"`
	CreditLimit  *decimal.Decimal `json:"credit_limit"`
	Balance      decimal.Decimal  `json:"balance"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

func toCustomerResponse(db *gorm.DB, customer models.Customer) CustomerResponse {
	balance, _ := services.CustomerBalance(db, customer.ID)
	return CustomerResponse{
		ID:           customer.ID,
		Name:         customer.Name,
		Phone:        customer.Phone,
		Address:      customer.Address,
		CustomerType: customer.CustomerType,
		Note:         customer.Note,
		IsActive:     customer.IsActive,
		CreditLimit:  customer.CreditLimit,
		Balance:      balance,
		CreatedAt:    customer.CreatedAt,
		UpdatedAt:    customer.UpdatedAt,
	}
}

type customerCreateRequest struct {
	Name        string           `json:"name"`
	Phone       string           `json:"phone"`
	Address     string           `json:"address"`
	Note        string           `json:"note"`
	IsActive    *bool            `json:"is_active"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
}

type customerUpdateRequest struct {
	Name        string           `json:"name"`
	Phone       string           `json:"phone"`
	Address     string           `json:"address"`
	Note        string           `json:"note"`
	IsActive    *bool            `json:"is_active"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
}

func checkAdminRole(c *gin.Context) bool {
	roleVal, exists := c.Get("role")
	if !exists {
		return false
	}
	role, _ := roleVal.(string)
	return strings.EqualFold(role, "admin")
}

// POST /api/customers
func (h *CustomerHandler) Create(c *gin.Context) {
	if !checkAdminRole(c) {
		fail(c, http.StatusForbidden, "forbidden: only administrators can create customers")
		return
	}

	var req customerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		fail(c, http.StatusBadRequest, "name is required")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if phone != "" {
		var count int64
		if err := h.DB.Model(&models.Customer{}).Where("phone = ?", phone).Count(&count).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if count > 0 {
			fail(c, http.StatusBadRequest, "Bu telefon numarasıyla kayıtlı bir cari müşteri zaten bulunuyor.")
			return
		}
	}

	if req.CreditLimit != nil && req.CreditLimit.IsNegative() {
		fail(c, http.StatusBadRequest, "credit_limit cannot be negative")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	customer := models.Customer{
		Name:         name,
		Phone:        phone,
		Address:      strings.TrimSpace(req.Address),
		CustomerType: "normal",
		Note:         strings.TrimSpace(req.Note),
		IsActive:     isActive,
		CreditLimit:  req.CreditLimit,
	}

	if err := h.DB.Create(&customer).Error; err != nil {
		handleDBError(c, err)
		return
	}

	created(c, toCustomerResponse(h.DB, customer))
}

// GET /api/customers
func (h *CustomerHandler) List(c *gin.Context) {
	isAdmin := checkAdminRole(c)

	query := h.DB.Model(&models.Customer{}).Order("id desc")

	// Cashiers / non-admins can only see active customers
	if !isAdmin {
		query = query.Where("is_active = ?", true)
	} else {
		if activeQuery := c.Query("is_active"); activeQuery != "" {
			if activeQuery == "true" || activeQuery == "1" {
				query = query.Where("is_active = ?", true)
			} else if activeQuery == "false" || activeQuery == "0" {
				query = query.Where("is_active = ?", false)
			}
		}
	}

	if q := strings.TrimSpace(strings.ToLower(c.Query("q"))); q != "" {
		query = query.Where("LOWER(name) LIKE ? OR phone LIKE ?", "%"+q+"%", "%"+q+"%")
	}

	var customers []models.Customer
	if err := query.Find(&customers).Error; err != nil {
		handleDBError(c, err)
		return
	}

	responses := make([]CustomerResponse, 0, len(customers))
	for _, cust := range customers {
		responses = append(responses, toCustomerResponse(h.DB, cust))
	}

	ok(c, responses)
}

// GET /api/customers/:id
func (h *CustomerHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var customer models.Customer
	if err := h.DB.First(&customer, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	// Non-admin cashier should not view inactive customer
	if !checkAdminRole(c) && !customer.IsActive {
		fail(c, http.StatusNotFound, "customer record not found")
		return
	}

	ok(c, toCustomerResponse(h.DB, customer))
}

// PUT /api/customers/:id
func (h *CustomerHandler) Update(c *gin.Context) {
	if !checkAdminRole(c) {
		fail(c, http.StatusForbidden, "forbidden: only administrators can edit customers")
		return
	}

	id, valid := parseID(c)
	if !valid {
		return
	}

	var customer models.Customer
	if err := h.DB.First(&customer, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var req customerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		fail(c, http.StatusBadRequest, "name is required")
		return
	}

	phone := strings.TrimSpace(req.Phone)
	if phone != "" && phone != customer.Phone {
		var count int64
		if err := h.DB.Model(&models.Customer{}).Where("phone = ? AND id <> ?", phone, id).Count(&count).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if count > 0 {
			fail(c, http.StatusBadRequest, "Bu telefon numarasıyla kayıtlı bir cari müşteri zaten bulunuyor.")
			return
		}
	}

	if req.CreditLimit != nil && req.CreditLimit.IsNegative() {
		fail(c, http.StatusBadRequest, "credit_limit cannot be negative")
		return
	}

	customer.Name = name
	customer.Phone = phone
	customer.Address = strings.TrimSpace(req.Address)
	customer.Note = strings.TrimSpace(req.Note)
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}
	customer.CreditLimit = req.CreditLimit

	if err := h.DB.Save(&customer).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, toCustomerResponse(h.DB, customer))
}

// DELETE /api/customers/:id
func (h *CustomerHandler) Delete(c *gin.Context) {
	if !checkAdminRole(c) {
		fail(c, http.StatusForbidden, "forbidden: only administrators can delete/deactivate customers")
		return
	}

	id, valid := parseID(c)
	if !valid {
		return
	}

	var customer models.Customer
	if err := h.DB.First(&customer, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	customer.IsActive = false
	if err := h.DB.Model(&customer).Update("is_active", false).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, gin.H{"deleted": true, "message": "customer deactivated successfully"})
}

// GET /api/customers/:id/balance
func (h *CustomerHandler) Balance(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var customer models.Customer
	if err := h.DB.First(&customer, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	balance, err := services.CustomerBalance(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"customer_id": id, "balance": balance})
}
