package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type StockMovementHandler struct{ DB *gorm.DB }

type stockMovementRequest struct {
	ProductID       uint            `json:"product_id"`
	ProductIDAlt    uint            `json:"productId"`
	MovementDate    string          `json:"movement_date"`
	MovementDateAlt string          `json:"movementDate"`
	Type            string          `json:"type"`
	MovementTypeAlt string          `json:"movementType"`
	Quantity        decimal.Decimal `json:"quantity"`
	UnitPrice       decimal.Decimal `json:"unit_price"`
	Note            string          `json:"note"`
}

type bulkStockMovementRequest struct {
	Entries []stockMovementRequest `json:"entries"`
}

func NewStockMovementHandler(db *gorm.DB) *StockMovementHandler {
	return &StockMovementHandler{DB: db}
}

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

	// Preload product info for response
	h.DB.Preload("Product").First(&movement, movement.ID)

	created(c, movement)
}

func (h *StockMovementHandler) BulkCreate(c *gin.Context) {
	var entries []stockMovementRequest

	// Try binding wrapper object `{ "entries": [...] }` first
	var bulkReq bulkStockMovementRequest
	if err := c.ShouldBindJSON(&bulkReq); err == nil && len(bulkReq.Entries) > 0 {
		entries = bulkReq.Entries
	} else {
		// Try binding raw array `[...]`
		var rawEntries []stockMovementRequest
		if err := c.ShouldBindJSON(&rawEntries); err == nil {
			entries = rawEntries
		}
	}

	if len(entries) == 0 {
		fail(c, http.StatusBadRequest, "entries list cannot be empty")
		return
	}

	var createdMovements []models.StockMovement

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range entries {
			movement, err := req.toModel()
			if err != nil {
				return err
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}
			tx.Preload("Product").First(&movement, movement.ID)
			createdMovements = append(createdMovements, movement)
		}
		return nil
	})

	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	created(c, createdMovements)
}

func (h *StockMovementHandler) List(c *gin.Context) {
	var movements []models.StockMovement
	query := h.DB.Preload("Product").Order("movement_date desc, id desc")

	productID := c.Query("product_id")
	if productID == "" {
		productID = c.Query("productId")
	}

	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}

	if err := query.Find(&movements).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, movements)
}

func (r stockMovementRequest) toModel() (models.StockMovement, error) {
	productID := r.ProductID
	if productID == 0 {
		productID = r.ProductIDAlt
	}
	if productID == 0 {
		return models.StockMovement{}, errRequired("product_id")
	}

	rawType := strings.TrimSpace(r.Type)
	if rawType == "" {
		rawType = strings.TrimSpace(r.MovementTypeAlt)
	}
	normalizedType := normalizeType(rawType)

	if !validateType(normalizedType, map[string]bool{"in": true, "out": true, "waste": true, "correction": true}) {
		return models.StockMovement{}, errInvalidType("type")
	}

	if err := positiveDecimal(r.Quantity, "quantity"); err != nil {
		return models.StockMovement{}, err
	}
	if err := notNegativeDecimal(r.UnitPrice, "unit_price"); err != nil {
		return models.StockMovement{}, err
	}

	dateStr := strings.TrimSpace(r.MovementDate)
	if dateStr == "" {
		dateStr = strings.TrimSpace(r.MovementDateAlt)
	}
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := parseDate(dateStr)
	if err != nil {
		return models.StockMovement{}, err
	}

	return models.StockMovement{
		ProductID:    productID,
		MovementDate: date,
		Type:         normalizedType,
		Quantity:     r.Quantity,
		UnitPrice:    r.UnitPrice,
		Note:         r.Note,
	}, nil
}

func normalizeType(t string) string {
	lower := strings.ToLower(t)
	switch lower {
	case "stock_in", "in", "return":
		return "in"
	case "stock_out", "out":
		return "out"
	case "adjustment", "manual_adjustment", "correction":
		return "correction"
	case "waste":
		return "waste"
	default:
		return lower
	}
}
