package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryHandler struct{ DB *gorm.DB }

type categoryRequest struct {
	Name string `json:"name"`
}

type categoryResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	IsActive     bool      `json:"is_active"`
	ProductCount int64     `json:"product_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler { return &CategoryHandler{DB: db} }

func (h *CategoryHandler) List(c *gin.Context) {
	var rows []categoryResponse
	err := h.DB.Raw(`
		SELECT
			c.id,
			c.name,
			c.is_active,
			c.created_at,
			c.updated_at,
			COUNT(p.id) AS product_count
		FROM categories c
		LEFT JOIN products p
			ON LOWER(BTRIM(p.category)) = LOWER(BTRIM(c.name))
		GROUP BY c.id, c.name, c.is_active, c.created_at, c.updated_at
		ORDER BY c.name ASC
	`).Scan(&rows).Error
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, rows)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Kategori bilgileri okunamadı.")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		fail(c, http.StatusBadRequest, "Kategori adı zorunludur.")
		return
	}

	var row categoryResponse
	err := h.DB.Raw(`
		INSERT INTO categories (name, is_active, created_at, updated_at)
		VALUES (?, TRUE, NOW(), NOW())
		RETURNING id, name, is_active, created_at, updated_at
	`, name).Scan(&row).Error
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, row)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var req categoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "Kategori bilgileri okunamadı.")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		fail(c, http.StatusBadRequest, "Kategori adı zorunludur.")
		return
	}

	var oldName string
	if err := h.DB.Raw("SELECT name FROM categories WHERE id = ?", id).Scan(&oldName).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if oldName == "" {
		fail(c, http.StatusNotFound, "Kategori bulunamadı.")
		return
	}

	var row categoryResponse
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"UPDATE products SET category = ? WHERE LOWER(BTRIM(category)) = LOWER(BTRIM(?))",
			name, oldName,
		).Error; err != nil {
			return err
		}
		return tx.Raw(`
			UPDATE categories
			SET name = ?, updated_at = NOW()
			WHERE id = ?
			RETURNING id, name, is_active, created_at, updated_at
		`, name, id).Scan(&row).Error
	})
	if err != nil {
		handleDBError(c, err)
		return
	}

	if err := h.DB.Raw(
		"SELECT COUNT(*) FROM products WHERE LOWER(BTRIM(category)) = LOWER(BTRIM(?))",
		row.Name,
	).Scan(&row.ProductCount).Error; err != nil {
		handleDBError(c, err)
		return
	}

	ok(c, row)
}
