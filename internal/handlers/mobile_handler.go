package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type MobileHandler struct {
	DB *gorm.DB
}

func NewMobileHandler(db *gorm.DB) *MobileHandler {
	return &MobileHandler{DB: db}
}

type MobileMoneyDto struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type MobileCategoryDto struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MobileProductDto struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Barcode      string            `json:"barcode"`
	ImageURL     string            `json:"imageUrl"`
	Category     MobileCategoryDto `json:"category"`
	CurrentPrice MobileMoneyDto    `json:"currentPrice"`
	Stock        decimal.Decimal   `json:"stock"`
	IsNew        bool              `json:"isNew"`
	IsBestseller bool              `json:"isBestseller"`
	CreatedAt    string            `json:"createdAt"`
}

type MobilePaginatedDto struct {
	Items    interface{} `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int64       `json:"total"`
}

// GET /api/mobile/products
func (h *MobileHandler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	queryStr := strings.TrimSpace(strings.ToLower(c.DefaultQuery("query", c.Query("q"))))
	categoryFilter := strings.TrimSpace(c.Query("categoryId"))

	var whereConditions []string
	var args []interface{}
	whereConditions = append(whereConditions, "p.is_active = true")

	if queryStr != "" {
		whereConditions = append(whereConditions, "(LOWER(p.name) LIKE ? OR p.barcode LIKE ?)")
		args = append(args, "%"+queryStr+"%", "%"+queryStr+"%")
	}
	if categoryFilter != "" {
		whereConditions = append(whereConditions, "p.category = ?")
		args = append(args, categoryFilter)
	}

	whereClause := "WHERE " + strings.Join(whereConditions, " AND ")

	var total int64
	countQuery := "SELECT COUNT(*) FROM products p " + whereClause
	if err := h.DB.Raw(countQuery, args...).Scan(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "products count failed"})
		return
	}

	type ProductRow struct {
		ID           uint
		Name         string
		Barcode      *string
		Category     string
		Brand        string
		Description  string
		ImageURL     string
		SalePrice    decimal.Decimal
		IsBestseller bool
		CreatedAt    string
		Stock        decimal.Decimal
	}

	selectQuery := `
		SELECT 
			p.id, 
			p.name, 
			p.barcode, 
			p.category, 
			p.brand, 
			p.description, 
			p.image_url, 
			p.sale_price, 
			p.is_bestseller, 
			p.created_at::text,
			COALESCE(SUM(CASE WHEN sm.type IN ('in', 'correction') THEN sm.quantity WHEN sm.type IN ('out', 'waste') THEN -sm.quantity ELSE 0 END), 0) AS stock
		FROM products p
		LEFT JOIN stock_movements sm ON sm.product_id = p.id
		` + whereClause + `
		GROUP BY p.id, p.name, p.barcode, p.category, p.brand, p.description, p.image_url, p.sale_price, p.is_bestseller, p.created_at
		ORDER BY p.name ASC
		LIMIT ? OFFSET ?
	`
	queryArgs := append(args, pageSize, offset)

	var rows []ProductRow
	if err := h.DB.Raw(selectQuery, queryArgs...).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "products fetch failed"})
		return
	}

	dtos := make([]MobileProductDto, 0, len(rows))
	for _, r := range rows {
		barcode := ""
		if r.Barcode != nil {
			barcode = *r.Barcode
		}
		img := r.ImageURL
		if img == "" {
			img = "https://placehold.co/600x600/png?text=" + r.Name
		}
		cat := r.Category
		if cat == "" {
			cat = "other"
		}

		dtos = append(dtos, MobileProductDto{
			ID:           strconv.FormatUint(uint64(r.ID), 10),
			Name:         r.Name,
			Barcode:      barcode,
			ImageURL:     img,
			Category:     MobileCategoryDto{ID: cat, Name: cat},
			CurrentPrice: MobileMoneyDto{Amount: r.SalePrice.StringFixed(2), Currency: "TRY"},
			Stock:        r.Stock,
			IsBestseller: r.IsBestseller,
			CreatedAt:    r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, MobilePaginatedDto{
		Items:    dtos,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// GET /api/mobile/products/:id
func (h *MobileHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	var product models.Product
	if err := h.DB.Where("id = ? OR barcode = ?", idStr, idStr).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	var stock decimal.Decimal
	h.DB.Raw(`
		SELECT COALESCE(SUM(CASE WHEN type IN ('in', 'correction') THEN quantity WHEN type IN ('out', 'waste') THEN -quantity ELSE 0 END), 0)
		FROM stock_movements WHERE product_id = ?
	`, product.ID).Scan(&stock)

	barcode := ""
	if product.Barcode != nil {
		barcode = *product.Barcode
	}
	img := product.ImageURL
	if img == "" {
		img = "https://placehold.co/600x600/png?text=" + product.Name
	}
	cat := product.Category
	if cat == "" {
		cat = "other"
	}

	c.JSON(http.StatusOK, MobileProductDto{
		ID:           strconv.FormatUint(uint64(product.ID), 10),
		Name:         product.Name,
		Barcode:      barcode,
		ImageURL:     img,
		Category:     MobileCategoryDto{ID: cat, Name: cat},
		CurrentPrice: MobileMoneyDto{Amount: product.SalePrice.StringFixed(2), Currency: "TRY"},
		Stock:        stock,
		IsBestseller: product.IsBestseller,
		CreatedAt:    product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GET /api/mobile/categories
func (h *MobileHandler) Categories(c *gin.Context) {
	type CategoryRow struct {
		Category string `json:"id"`
	}
	var rows []CategoryRow
	h.DB.Raw("SELECT DISTINCT category FROM products WHERE category != '' AND is_active = true ORDER BY category ASC").Scan(&rows)

	result := make([]MobileCategoryDto, 0, len(rows))
	for _, r := range rows {
		result = append(result, MobileCategoryDto{ID: r.Category, Name: r.Category})
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/mobile/customer/profile
func (h *MobileHandler) GetCustomerProfile(c *gin.Context) {
	phoneStr := strings.TrimSpace(c.Query("phone"))
	if phoneStr == "" {
		c.JSON(http.StatusOK, gin.H{"has_customer": false, "customer": nil})
		return
	}

	var customer models.Customer
	if err := h.DB.Where("(phone = ? OR REPLACE(phone, ' ', '') = REPLACE(?, ' ', '')) AND is_active = true", phoneStr, phoneStr).First(&customer).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"has_customer": false, "customer": nil})
		return
	}

	balance, _ := services.CustomerBalance(h.DB, customer.ID)

	var lastPurchaseAt *string
	var lastSaleDate string

	dateCast := "created_at::text"
	if h.DB.Dialector.Name() == "sqlite" {
		dateCast = "created_at"
	}
	if err := h.DB.Raw(`
		SELECT ` + dateCast + ` FROM sales WHERE created_by IN (SELECT id FROM users WHERE phone = ?) OR id IN (SELECT sale_id FROM customer_transactions WHERE customer_id = ? AND sale_id IS NOT NULL) ORDER BY created_at DESC LIMIT 1
	`, phoneStr, customer.ID).Scan(&lastSaleDate).Error; err == nil && lastSaleDate != "" {
		lastPurchaseAt = &lastSaleDate
	}

	c.JSON(http.StatusOK, gin.H{
		"has_customer": true,
		"customer": gin.H{
			"id":               customer.ID,
			"name":             customer.Name,
			"phone":            customer.Phone,
			"address":          customer.Address,
			"note":             customer.Note,
			"is_active":        customer.IsActive,
			"credit_limit":     customer.CreditLimit,
			"balance":          balance,
			"last_purchase_at": lastPurchaseAt,
			"created_at":       customer.CreatedAt,
		},
	})
}

// GET /api/mobile/customer/transactions
func (h *MobileHandler) GetCustomerTransactions(c *gin.Context) {
	phoneStr := strings.TrimSpace(c.Query("phone"))
	if phoneStr == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var customer models.Customer
	if err := h.DB.Where("(phone = ? OR REPLACE(phone, ' ', '') = REPLACE(?, ' ', '')) AND is_active = true", phoneStr, phoneStr).First(&customer).Error; err != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var txs []models.CustomerTransaction
	if err := h.DB.Where("customer_id = ?", customer.ID).Order("transaction_date desc, id desc").Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch transactions failed"})
		return
	}

	c.JSON(http.StatusOK, txs)
}
