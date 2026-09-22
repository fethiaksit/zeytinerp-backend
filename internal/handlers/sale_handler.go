package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type SaleHandler struct{ DB *gorm.DB }

func NewSaleHandler(db *gorm.DB) *SaleHandler { return &SaleHandler{DB: db} }

type saleItemRequest struct {
	ProductID uint            `json:"product_id"`
	Barcode   string          `json:"barcode"`
	Quantity  decimal.Decimal `json:"quantity"`
}

type saleCreateRequest struct {
	PaymentMethod string            `json:"payment_method"`
	CustomerID    *uint             `json:"customer_id"`
	Items         []saleItemRequest `json:"items"`
}

type saleItemResponse struct {
	ID          uint            `json:"id"`
	ProductID   uint            `json:"product_id"`
	Barcode     string          `json:"barcode"`
	ProductName string          `json:"product_name"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	LineTotal   decimal.Decimal `json:"line_total"`
}

type saleResponse struct {
	ID                 uint               `json:"id"`
	SaleNo             string             `json:"sale_no"`
	PaymentMethod      string             `json:"payment_method"`
	CustomerID         *uint              `json:"customer_id"`
	TotalAmount        decimal.Decimal    `json:"total_amount"`
	CustomerNewBalance *decimal.Decimal   `json:"customer_new_balance,omitempty"`
	Items              []saleItemResponse `json:"items"`
	CreatedAt          time.Time          `json:"created_at"`
}

func (h *SaleHandler) Create(c *gin.Context) {
	var req saleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Satış bilgileri okunamadı.")
		return
	}

	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if req.PaymentMethod != "cash" && req.PaymentMethod != "card" && req.PaymentMethod != "current" {
		fail(c, http.StatusBadRequest, "Geçersiz ödeme türü.")
		return
	}
	if len(req.Items) == 0 {
		fail(c, http.StatusBadRequest, "Sepette ürün bulunmuyor.")
		return
	}
	if req.PaymentMethod == "current" && (req.CustomerID == nil || *req.CustomerID == 0) {
		fail(c, http.StatusBadRequest, "Cari satış için müşteri seçilmelidir.")
		return
	}

	var result saleResponse
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if req.PaymentMethod == "current" {
			var customer models.Customer
			if err := tx.First(&customer, *req.CustomerID).Error; err != nil {
				return fmt.Errorf("cari müşteri bulunamadı")
			}
			if !customer.IsActive {
				return fmt.Errorf("seçilen cari müşteri pasif durumda")
			}
		}

		saleNo := fmt.Sprintf("POS-%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000000)

		var createdBy interface{} = nil
		if userID, exists := c.Get("user_id"); exists {
			createdBy = userID
		}

		var saleID uint
		if err := tx.Raw(
			"INSERT INTO sales (sale_no, payment_method, customer_id, total_amount, created_by, created_at) VALUES (?, ?, ?, 0, ?, NOW()) RETURNING id",
			saleNo, req.PaymentMethod, req.CustomerID, createdBy,
		).Scan(&saleID).Error; err != nil {
			return err
		}

		total := decimal.Zero
		items := make([]saleItemResponse, 0, len(req.Items))

		for _, item := range req.Items {
			if item.ProductID == 0 || !item.Quantity.GreaterThan(decimal.Zero) {
				return fmt.Errorf("ürün miktarı sıfırdan büyük olmalıdır")
			}

			var product models.Product
			if err := tx.First(&product, item.ProductID).Error; err != nil {
				return fmt.Errorf("ürün bulunamadı")
			}
			if !product.IsActive {
				return fmt.Errorf("%s satışa kapalı", product.Name)
			}

			stock, err := services.ProductStock(tx, product.ID)
			if err != nil {
				return err
			}
			if stock.LessThan(item.Quantity) {
				return fmt.Errorf("%s için stok yetersiz. Mevcut stok: %s", product.Name, stock.String())
			}

			barcode := ""
			if product.Barcode != nil {
				barcode = *product.Barcode
			}
			lineTotal := product.SalePrice.Mul(item.Quantity)
			total = total.Add(lineTotal)

			var itemID uint
			if err := tx.Raw(
				"INSERT INTO sale_items (sale_id, product_id, barcode, product_name, quantity, unit_price, line_total) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id",
				saleID, product.ID, barcode, product.Name, item.Quantity, product.SalePrice, lineTotal,
			).Scan(&itemID).Error; err != nil {
				return err
			}

			movement := models.StockMovement{
				ProductID:    product.ID,
				MovementDate: time.Now(),
				Type:         "out",
				Quantity:     item.Quantity,
				UnitPrice:    product.SalePrice,
				Note:         "POS satış: " + saleNo,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}

			items = append(items, saleItemResponse{
				ID: itemID, ProductID: product.ID, Barcode: barcode,
				ProductName: product.Name, Quantity: item.Quantity,
				UnitPrice: product.SalePrice, LineTotal: lineTotal,
			})
		}

		if err := tx.Exec("UPDATE sales SET total_amount = ? WHERE id = ?", total, saleID).Error; err != nil {
			return err
		}

		var newBalance *decimal.Decimal
		if req.PaymentMethod == "current" && req.CustomerID != nil {
			txRow := models.CustomerTransaction{
				CustomerID:      *req.CustomerID,
				TransactionDate: time.Now(),
				Type:            "debt",
				Amount:          total,
				SaleID:          &saleID,
				Note:            "Cari alışveriş - " + saleNo,
			}
			if err := tx.Create(&txRow).Error; err != nil {
				return err
			}

			balance, err := services.CustomerBalance(tx, *req.CustomerID)
			if err != nil {
				return err
			}
			newBalance = &balance
		}

		result = saleResponse{
			ID: saleID, SaleNo: saleNo, PaymentMethod: req.PaymentMethod,
			CustomerID: req.CustomerID, TotalAmount: total,
			CustomerNewBalance: newBalance, Items: items, CreatedAt: time.Now(),
		}
		return nil
	})
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	created(c, result)
}

func (h *SaleHandler) Get(c *gin.Context) {
	key := strings.TrimSpace(c.Param("id"))
	if key == "" {
		fail(c, http.StatusBadRequest, "Satış numarası zorunludur.")
		return
	}

	var row struct {
		ID            uint
		SaleNo        string
		PaymentMethod string
		CustomerID    *uint
		TotalAmount   decimal.Decimal
		CreatedAt     time.Time
	}
	if err := h.DB.Raw(
		"SELECT id, sale_no, payment_method, customer_id, total_amount, created_at FROM sales WHERE CAST(id AS TEXT) = ? OR sale_no = ? LIMIT 1",
		key, key,
	).Scan(&row).Error; err != nil || row.ID == 0 {
		fail(c, http.StatusNotFound, "Satış kaydı bulunamadı.")
		return
	}

	var items []saleItemResponse
	if err := h.DB.Raw(
		"SELECT id, product_id, barcode, product_name, quantity, unit_price, line_total FROM sale_items WHERE sale_id = ? ORDER BY id ASC",
		row.ID,
	).Scan(&items).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, saleResponse{
		ID: row.ID, SaleNo: row.SaleNo, PaymentMethod: row.PaymentMethod,
		CustomerID: row.CustomerID, TotalAmount: row.TotalAmount,
		Items: items, CreatedAt: row.CreatedAt,
	})
}
