package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type AdminEmployeeHandler struct {
	DB *gorm.DB
}

func NewAdminEmployeeHandler(db *gorm.DB) *AdminEmployeeHandler {
	return &AdminEmployeeHandler{DB: db}
}

type AdminEmployeeResponse struct {
	ID          uint       `json:"id"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	FullName    string     `json:"full_name"`
	Phone       *string    `json:"phone"`
	Username    string     `json:"username"`
	Role        string     `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toAdminEmployeeResponse(user models.User) AdminEmployeeResponse {
	return AdminEmployeeResponse{
		ID:          user.ID,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		FullName:    user.FullName(),
		Phone:       user.Phone,
		Username:    user.Username,
		Role:        user.Role,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
	}
}

func toAdminEmployeeResponseList(users []models.User) []AdminEmployeeResponse {
	res := make([]AdminEmployeeResponse, 0, len(users))
	for _, u := range users {
		res = append(res, toAdminEmployeeResponse(u))
	}
	return res
}

type createAdminEmployeeRequest struct {
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Phone           string `json:"phone"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Role            string `json:"role"`
	IsActive        *bool  `json:"is_active"`
}

type updateAdminEmployeeRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Username  string `json:"username"`
	Role      string `json:"role"`
}

type updateAdminEmployeeStatusRequest struct {
	IsActive bool `json:"is_active"`
}

type updateAdminEmployeePasswordRequest struct {
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func isValidRole(role string) bool {
	switch role {
	case "admin", "cashier", "warehouse":
		return true
	default:
		return false
	}
}

// GET /api/admin/employees
func (h *AdminEmployeeHandler) List(c *gin.Context) {
	var users []models.User
	if err := h.DB.Order("id asc").Find(&users).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, toAdminEmployeeResponseList(users))
}

// POST /api/admin/employees
func (h *AdminEmployeeHandler) Create(c *gin.Context) {
	var req createAdminEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		fail(c, http.StatusBadRequest, "username is required")
		return
	}

	role := strings.TrimSpace(req.Role)
	if !isValidRole(role) {
		fail(c, http.StatusBadRequest, "invalid role: must be admin, cashier, or warehouse")
		return
	}

	if len(req.Password) < 6 {
		fail(c, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	if req.Password != req.ConfirmPassword {
		fail(c, http.StatusBadRequest, "password and confirm_password must match")
		return
	}

	// Check unique username
	var count int64
	if err := h.DB.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if count > 0 {
		fail(c, http.StatusBadRequest, "username already exists")
		return
	}

	// Check unique phone if provided
	phoneStr := strings.TrimSpace(req.Phone)
	var phonePtr *string
	if phoneStr != "" {
		if err := h.DB.Model(&models.User{}).Where("phone = ?", phoneStr).Count(&count).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if count > 0 {
			fail(c, http.StatusBadRequest, "phone number already in use")
			return
		}
		phonePtr = &phoneStr
	}

	passwordHash, err := services.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)

	user := models.User{
		Username:     username,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
		Name:         strings.TrimSpace(firstName + " " + lastName),
		Phone:        phonePtr,
		Role:         role,
		IsActive:     isActive,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		handleDBError(c, err)
		return
	}

	created(c, toAdminEmployeeResponse(user))
}

// PUT /api/admin/employees/:id
func (h *AdminEmployeeHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var req updateAdminEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		fail(c, http.StatusBadRequest, "username is required")
		return
	}

	role := strings.TrimSpace(req.Role)
	if !isValidRole(role) {
		fail(c, http.StatusBadRequest, "invalid role: must be admin, cashier, or warehouse")
		return
	}

	if username != user.Username {
		var count int64
		if err := h.DB.Model(&models.User{}).Where("username = ? AND id <> ?", username, id).Count(&count).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if count > 0 {
			fail(c, http.StatusBadRequest, "username already exists")
			return
		}
	}

	phoneStr := strings.TrimSpace(req.Phone)
	var phonePtr *string
	if phoneStr != "" {
		var count int64
		if err := h.DB.Model(&models.User{}).Where("phone = ? AND id <> ?", phoneStr, id).Count(&count).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if count > 0 {
			fail(c, http.StatusBadRequest, "phone number already in use")
			return
		}
		phonePtr = &phoneStr
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)

	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.Name = strings.TrimSpace(firstName + " " + lastName)
	user.Phone = phonePtr
	user.Role = role

	if err := h.DB.Save(&user).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, toAdminEmployeeResponse(user))
}

// PATCH /api/admin/employees/:id/status
func (h *AdminEmployeeHandler) UpdateStatus(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var req updateAdminEmployeeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	user.IsActive = req.IsActive
	if err := h.DB.Model(&user).Update("is_active", req.IsActive).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, toAdminEmployeeResponse(user))
}

// PATCH /api/admin/employees/:id/password
func (h *AdminEmployeeHandler) UpdatePassword(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var req updateAdminEmployeePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	if len(req.Password) < 6 {
		fail(c, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	if req.Password != req.ConfirmPassword {
		fail(c, http.StatusBadRequest, "password and confirm_password must match")
		return
	}

	passwordHash, err := services.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user.PasswordHash = passwordHash
	if err := h.DB.Model(&user).Update("password_hash", passwordHash).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, gin.H{"message": "password updated successfully"})
}

// DELETE /api/admin/employees/:id
func (h *AdminEmployeeHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		handleDBError(c, err)
		return
	}

	user.IsActive = false
	if err := h.DB.Model(&user).Update("is_active", false).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, gin.H{"deleted": true, "message": "user deactivated successfully"})
}
