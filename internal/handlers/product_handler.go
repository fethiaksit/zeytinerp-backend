package handlers

import (
	"errors"
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

type productFavoriteRequest struct {
	IsFavorite bool `json:"isFavorite"`
}

var errFavoriteLimit = errors.New("En fazla 10 favori ürün seçebilirsiniz.")

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

type bulkImportItem struct {
	LineNumber    int             `json:"lineNumber"`
	Barcode       string          `json:"barcode"`
	Name          string          `json:"name"`
	Price         decimal.Decimal `json:"price"`
	PurchasePrice decimal.Decimal `json:"purchasePrice"`
	Stock         decimal.Decimal `json:"stock"`
	Category      string          `json:"category"`
	Unit          string          `json:"unit"`
	Status        string          `json:"status"`
}

type bulkImportRequest struct {
	Items          []bulkImportItem `json:"items"`
	ExistingAction string           `json:"existingAction"`
	CreatedBy      string           `json:"createdBy"`
}

type bulkImportResponse struct {
	Created    int `json:"created"`
	Updated    int `json:"updated"`
	Skipped    int `json:"skipped"`
	StockAdded int `json:"stockAdded"`
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

func (h *ProductHandler) BulkImport(c *gin.Context) {
	var req bulkImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "CSV aktarım verisi okunamadı.")
		return
	}
	if len(req.Items) == 0 {
		fail(c, http.StatusBadRequest, "Aktarılacak ürün bulunamadı.")
		return
	}

	action := strings.ToUpper(strings.TrimSpace(req.ExistingAction))
	if action == "" {
		action = "UPDATE_INFO"
	}
	if action != "UPDATE_INFO" && action != "ADD_STOCK_ONLY" && action != "SKIP" {
		fail(c, http.StatusBadRequest, "Geçersiz mevcut ürün işlemi.")
		return
	}

	result := bulkImportResponse{}
	today := time.Now()

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if strings.EqualFold(strings.TrimSpace(item.Status), "ERROR") {
				result.Skipped++
				continue
			}

			barcode := strings.TrimSpace(item.Barcode)
			name := strings.TrimSpace(item.Name)
			category := strings.TrimSpace(item.Category)

			if barcode == "" || name == "" {
				result.Skipped++
				continue
			}
			if item.Price.IsNegative() || item.PurchasePrice.IsNegative() || item.Stock.IsNegative() {
				return errInvalidType("CSV fiyat/stok")
			}

			var product models.Product
			findErr := tx.Where("barcode = ?", barcode).First(&product).Error

			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				active := true
				product = models.Product{
					Name:          name,
					Barcode:       &barcode,
					Category:      category,
					PurchasePrice: item.PurchasePrice,
					SalePrice:     item.Price,
					CriticalStock: decimal.Zero,
					IsActive:      active,
				}
				if err := tx.Create(&product).Error; err != nil {
					return err
				}

				if item.Stock.GreaterThan(decimal.Zero) {
					movement := models.StockMovement{
						ProductID:    product.ID,
						MovementDate: today,
						Type:         "in",
						Quantity:     item.Stock,
						UnitPrice:    item.PurchasePrice,
						Note:         "CSV ile ilk stok girişi",
					}
					if err := tx.Create(&movement).Error; err != nil {
						return err
					}
					result.StockAdded++
				}
				result.Created++
				continue
			}
			if findErr != nil {
				return findErr
			}

			switch action {
			case "SKIP":
				result.Skipped++

			case "UPDATE_INFO":
				updates := map[string]interface{}{
					"name":       name,
					"sale_price": item.Price,
					"category":   category,
					"updated_at": time.Now(),
				}
				if !item.PurchasePrice.IsZero() {
					updates["purchase_price"] = item.PurchasePrice
				}
				if err := tx.Model(&product).Updates(updates).Error; err != nil {
					return err
				}
				result.Updated++

			case "ADD_STOCK_ONLY":
				if item.Stock.GreaterThan(decimal.Zero) {
					movement := models.StockMovement{
						ProductID:    product.ID,
						MovementDate: today,
						Type:         "in",
						Quantity:     item.Stock,
						UnitPrice:    item.PurchasePrice,
						Note:         "CSV ile toplu stok girişi",
					}
					if err := tx.Create(&movement).Error; err != nil {
						return err
					}
					result.StockAdded++
				} else {
					result.Skipped++
				}
			}
		}
		return nil
	})
	if err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, result)
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

func (h *ProductHandler) ToggleFavorite(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var req productFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var product models.Product
		if err := tx.Where("id = ? AND is_active = ?", id, true).First(&product).Error; err != nil {
			return err
		}

		if req.IsFavorite {
			if product.IsBestseller {
				return nil
			}

			var favoriteCount int64
			if err := tx.Model(&models.Product{}).
				Where("is_active = ? AND is_bestseller = ?", true, true).
				Count(&favoriteCount).Error; err != nil {
				return err
			}
			if favoriteCount >= 10 {
				return errFavoriteLimit
			}
			var nextOrder int
			if err := tx.Model(&models.Product{}).
				Select("COALESCE(MAX(bestseller_order), 0)").
				Scan(&nextOrder).Error; err != nil {
				return err
			}

			return tx.Model(&product).Updates(map[string]interface{}{
				"is_bestseller":    true,
				"bestseller_order": nextOrder + 1,
			}).Error
		}

		return tx.Model(&product).Updates(map[string]interface{}{
			"is_bestseller":    false,
			"bestseller_order": 0,
		}).Error
	})
	if errors.Is(err, errFavoriteLimit) {
		fail(c, http.StatusConflict, errFavoriteLimit.Error())
		return
	}
	if err != nil {
		handleDBError(c, err)
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
