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

type ProductHandler struct{ DB *gorm.DB }

type productRequest struct {
	Name          string          `json:"name"`
	Barcode       string          `json:"barcode"`
	Category      string          `json:"category"`
	PurchasePrice decimal.Decimal `json:"purchase_price"`
	SalePrice     decimal.Decimal `json:"sale_price"`
	CriticalStock decimal.Decimal `json:"critical_stock"`
	IsActive      *bool           `json:"is_active"`
}

type productResponse struct {
	ID              uint            `json:"id"`
	Name            string          `json:"name"`
	Barcode         *string         `json:"barcode"`
	Category        string          `json:"category"`
	Brand           string          `json:"brand"`
	Description     string          `json:"description"`
	ImageURL        string          `json:"image_url"`
	IsBestseller    bool            `json:"is_bestseller"`
	BestsellerOrder int             `json:"bestseller_order"`
	PurchasePrice   decimal.Decimal `json:"purchase_price"`
	SalePrice       decimal.Decimal `json:"sale_price"`
	CriticalStock   decimal.Decimal `json:"critical_stock"`
	Stock           decimal.Decimal `json:"stock"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func NewProductHandler(db *gorm.DB) *ProductHandler { return &ProductHandler{DB: db} }

func (h *ProductHandler) toResponse(product models.Product) (productResponse, error) {
	stock, err := services.ProductStock(h.DB, product.ID)
	if err != nil {
		return productResponse{}, err
	}
	return productResponse{
		ID: product.ID, Name: product.Name, Barcode: product.Barcode,
		Category: product.Category, Brand: product.Brand, Description: product.Description,
		ImageURL: product.ImageURL, IsBestseller: product.IsBestseller,
		BestsellerOrder: product.BestsellerOrder, PurchasePrice: product.PurchasePrice,
		SalePrice: product.SalePrice, CriticalStock: product.CriticalStock, Stock: stock,
		IsActive: product.IsActive, CreatedAt: product.CreatedAt, UpdatedAt: product.UpdatedAt,
	}, nil
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	product, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&product).Error; err != nil {
		handleDBError(c, err)
		return
	}
	resp, err := h.toResponse(product)
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, resp)
}

func (h *ProductHandler) List(c *gin.Context) {
	var products []models.Product
	if err := h.DB.Order("id desc").Find(&products).Error; err != nil {
		handleDBError(c, err)
		return
	}
	responses := make([]productResponse, 0, len(products))
	for _, product := range products {
		resp, err := h.toResponse(product)
		if err != nil {
			handleDBError(c, err)
			return
		}
		responses = append(responses, resp)
	}
	ok(c, responses)
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	resp, err := h.toResponse(product)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, resp)
}

func (h *ProductHandler) GetByBarcode(c *gin.Context) {
	barcode := strings.TrimSpace(c.Param("barcode"))
	if barcode == "" {
		fail(c, http.StatusBadRequest, "Barkod zorunludur.")
		return
	}
	var product models.Product
	if err := h.DB.Where("barcode = ?", barcode).First(&product).Error; err != nil {
		handleDBError(c, err)
		return
	}
	resp, err := h.toResponse(product)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, resp)
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	product, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.Product
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	product.ID = existing.ID
	product.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&product).Error; err != nil {
		handleDBError(c, err)
		return
	}
	resp, err := h.toResponse(product)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, resp)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Product{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (h *ProductHandler) Stock(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var product models.Product
	if err := h.DB.First(&product, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	stock, err := services.ProductStock(h.DB, id)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"product_id": id, "stock": stock})
}

func (r productRequest) toModel() (models.Product, error) {
	if strings.TrimSpace(r.Name) == "" {
		return models.Product{}, errRequired("name")
	}
	if err := notNegativeDecimal(r.PurchasePrice, "purchase_price"); err != nil {
		return models.Product{}, err
	}
	if err := notNegativeDecimal(r.SalePrice, "sale_price"); err != nil {
		return models.Product{}, err
	}
	if err := notNegativeDecimal(r.CriticalStock, "critical_stock"); err != nil {
		return models.Product{}, err
	}
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	var barcode *string
	if strings.TrimSpace(r.Barcode) != "" {
		clean := strings.TrimSpace(r.Barcode)
		barcode = &clean
	}
	return models.Product{Name: r.Name, Barcode: barcode, Category: r.Category, PurchasePrice: r.PurchasePrice, SalePrice: r.SalePrice, CriticalStock: r.CriticalStock, IsActive: active}, nil
}
