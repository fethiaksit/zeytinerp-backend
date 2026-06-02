package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type StockMovementHandler struct{ DB *gorm.DB }

type stockMovementRequest struct {
	ProductID    uint            `json:"product_id"`
	MovementDate string          `json:"movement_date"`
	Type         string          `json:"type"`
	Quantity     decimal.Decimal `json:"quantity"`
	UnitPrice    decimal.Decimal `json:"unit_price"`
	Note         string          `json:"note"`
}

func NewStockMovementHandler(db *gorm.DB) *StockMovementHandler { return &StockMovementHandler{DB: db} }

func (h *StockMovementHandler) Create(c *gin.Context) {
	var req stockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	movement, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&movement).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, movement)
}

func (h *StockMovementHandler) List(c *gin.Context) {
	var movements []models.StockMovement
	query := h.DB.Order("movement_date desc, id desc")
	if productID := c.Query("product_id"); productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	if err := query.Find(&movements).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, movements)
}

func (r stockMovementRequest) toModel() (models.StockMovement, error) {
	if r.ProductID == 0 {
		return models.StockMovement{}, errRequired("product_id")
	}
	if !validateType(r.Type, map[string]bool{"in": true, "out": true, "waste": true, "correction": true}) {
		return models.StockMovement{}, errInvalidType("type")
	}
	if err := positiveDecimal(r.Quantity, "quantity"); err != nil {
		return models.StockMovement{}, err
	}
	if err := notNegativeDecimal(r.UnitPrice, "unit_price"); err != nil {
		return models.StockMovement{}, err
	}
	date, err := parseDate(r.MovementDate)
	if err != nil {
		return models.StockMovement{}, err
	}
	return models.StockMovement{ProductID: r.ProductID, MovementDate: date, Type: r.Type, Quantity: r.Quantity, UnitPrice: r.UnitPrice, Note: r.Note}, nil
}
