package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type CustomerHandler struct{ DB *gorm.DB }

func NewCustomerHandler(db *gorm.DB) *CustomerHandler { return &CustomerHandler{DB: db} }

func (h *CustomerHandler) Create(c *gin.Context) {
	var customer models.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateCustomer(&customer); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&customer).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, customer)
}

func (h *CustomerHandler) List(c *gin.Context) {
	var customers []models.Customer
	if err := h.DB.Order("id desc").Find(&customers).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, customers)
}

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
	ok(c, customer)
}

func (h *CustomerHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var customer models.Customer
	if err := h.DB.First(&customer, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if err := c.ShouldBindJSON(&customer); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateCustomer(&customer); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	customer.ID = id
	if err := h.DB.Save(&customer).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, customer)
}

func (h *CustomerHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Customer{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

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

func validateCustomer(customer *models.Customer) error {
	if strings.TrimSpace(customer.Name) == "" {
		return errRequired("name")
	}
	if customer.CustomerType == "" {
		customer.CustomerType = "normal"
	}
	if !validateType(customer.CustomerType, map[string]bool{"normal": true, "veresiye": true, "tup": true}) {
		return errInvalidType("customer_type")
	}
	return nil
}
