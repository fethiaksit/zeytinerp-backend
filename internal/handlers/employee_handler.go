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

type EmployeeHandler struct{ DB *gorm.DB }

type employeeRequest struct {
	Name      string          `json:"name"`
	Phone     string          `json:"phone"`
	DailyWage decimal.Decimal `json:"daily_wage"`
	IsActive  *bool           `json:"is_active"`
	Note      string          `json:"note"`
}

func NewEmployeeHandler(db *gorm.DB) *EmployeeHandler { return &EmployeeHandler{DB: db} }

func (h *EmployeeHandler) Create(c *gin.Context) {
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	employee, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&employee).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, employee)
}

func (h *EmployeeHandler) List(c *gin.Context) {
	var employees []models.Employee
	if err := h.DB.Order("id desc").Find(&employees).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, employees)
}

func (h *EmployeeHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var employee models.Employee
	if err := h.DB.First(&employee, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, employee)
}

func (h *EmployeeHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req employeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	employee, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.Employee
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	employee.ID = existing.ID
	employee.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&employee).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, employee)
}

func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Employee{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *EmployeeHandler) Balance(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var employee models.Employee
	if err := h.DB.First(&employee, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	balance, err := services.EmployeeBalance(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"employee_id": id, "balance": balance})
}

func (h *EmployeeHandler) Balances(c *gin.Context) {
	balances, err := services.EmployeeBalances(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, balances)
}

func (r employeeRequest) toModel() (models.Employee, error) {
	if strings.TrimSpace(r.Name) == "" {
		return models.Employee{}, errRequired("name")
	}
	if err := positiveDecimal(r.DailyWage, "daily_wage"); err != nil {
		return models.Employee{}, err
	}
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return models.Employee{
		Name:      strings.TrimSpace(r.Name),
		Phone:     strings.TrimSpace(r.Phone),
		DailyWage: r.DailyWage,
		IsActive:  active,
		Note:      strings.TrimSpace(r.Note),
	}, nil
}
