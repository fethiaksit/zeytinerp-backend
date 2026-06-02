package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type AuthHandler struct {
	DB        *gorm.DB
	JWTSecret string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authUserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func NewAuthHandler(db *gorm.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: jwtSecret}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		fail(c, http.StatusBadRequest, errRequired("username").Error())
		return
	}
	if req.Password == "" {
		fail(c, http.StatusBadRequest, errRequired("password").Error())
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err := services.ComparePassword(user.PasswordHash, req.Password); err != nil {
		fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := services.GenerateJWT(h.JWTSecret, services.NewAuthClaims(user.ID, user.Username, user.Role))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{
		"token": token,
		"user":  authUser(user),
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := services.UserIDFromContextValue(c.GetUint("user_id"))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid token")
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, authUser(user))
}

func authUser(user models.User) authUserResponse {
	return authUserResponse{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
		Role:     user.Role,
	}
}
