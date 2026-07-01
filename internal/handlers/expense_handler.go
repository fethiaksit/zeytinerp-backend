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

type ExpenseHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

type expenseRequest struct {
	ExpenseDate   string          `json:"expense_date"`
	Category      string          `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	Note          string          `json:"note"`
}

type expensesByDateResponse struct {
	Success  bool                `json:"success"`
	Date     string              `json:"date"`
	Total    decimal.Decimal     `json:"total"`
	Count    int                 `json:"count"`
	Expenses []expenseByDateItem `json:"expenses"`
}

type expenseByDateItem struct {
	ID            uint            `json:"id"`
	ExpenseDate   time.Time       `json:"expense_date"`
	Category      string          `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	Note          string          `json:"note"`
}

func NewExpenseHandler(db *gorm.DB) *ExpenseHandler {
	return &ExpenseHandler{DB: db, Now: time.Now}
}

func (h *ExpenseHandler) Create(c *gin.Context) {
	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	expense, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&expense).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, expense)
}

func (h *ExpenseHandler) List(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	dateRange, valid := parseDateRange(c, startDate, endDate)
	if !valid {
		return
	}

	var expenses []models.Expense
	listQuery := applyExpenseDateRange(h.DB.Model(&models.Expense{}), dateRange)
	if err := listQuery.Order("expense_date desc, id desc").Find(&expenses).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var total decimal.Decimal
	totalQuery := applyExpenseDateRange(h.DB.Model(&models.Expense{}), dateRange)
	if err := totalQuery.Select("COALESCE(SUM(amount), 0)").Scan(&total).Error; err != nil {
		handleDBError(c, err)
		return
	}

	okWithTotalAmount(c, expenses, total)
}

func (h *ExpenseHandler) ListByDate(c *gin.Context) {
	dateValue, provided := c.GetQuery("date")
	dateValue = strings.TrimSpace(dateValue)
	if !provided {
		now := time.Now()
		if h.Now != nil {
			now = h.Now()
		}
		location, err := time.LoadLocation("Europe/Istanbul")
		if err != nil {
			location = time.FixedZone("Europe/Istanbul", 3*60*60)
		}
		dateValue = now.In(location).Format("2006-01-02")
	} else if dateValue == "" {
		fail(c, http.StatusBadRequest, "date is required")
		return
	}

	dayStart, err := time.Parse("2006-01-02", dateValue)
	if err != nil {
		fail(c, http.StatusBadRequest, "date is invalid; expected format is YYYY-MM-DD")
		return
	}
	dayEnd := dayStart.AddDate(0, 0, 1)

	expenses := make([]models.Expense, 0)
	if err := expensesForDayQuery(h.DB, dayStart, dayEnd).
		Order("expense_date desc, id desc").
		Find(&expenses).Error; err != nil {
		handleDBError(c, err)
		return
	}

	total := decimal.Zero
	items := make([]expenseByDateItem, 0, len(expenses))
	for _, expense := range expenses {
		total = total.Add(expense.Amount)
		items = append(items, expenseByDateItem{
			ID:            expense.ID,
			ExpenseDate:   expense.ExpenseDate,
			Category:      expense.Category,
			Amount:        expense.Amount,
			PaymentMethod: expense.PaymentMethod,
			Note:          expense.Note,
		})
	}

	c.JSON(http.StatusOK, expensesByDateResponse{
		Success:  true,
		Date:     dateValue,
		Total:    total,
		Count:    len(expenses),
		Expenses: items,
	})
}

func expensesForDayQuery(db *gorm.DB, dayStart, dayEnd time.Time) *gorm.DB {
	return db.Model(&models.Expense{}).
		Select("id, expense_date, category, amount, payment_method, note").
		Where("expense_date >= ? AND expense_date < ?", dayStart, dayEnd)
}

func (h *ExpenseHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	expense, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.Expense
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	expense.ID = existing.ID
	expense.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&expense).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, expense)
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.Expense{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r expenseRequest) toModel() (models.Expense, error) {
	if strings.TrimSpace(r.Category) == "" {
		return models.Expense{}, errRequired("category")
	}
	if !validateType(r.Category, map[string]bool{
		"kira":          true,
		"elektrik":      true,
		"su":            true,
		"personel":      true,
		"yakit":         true,
		"yemek":         true,
		"market_gideri": true,
		"diger":         true,
	}) {
		return models.Expense{}, errInvalidType("category")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.Expense{}, err
	}
	date, err := parseDate(r.ExpenseDate)
	if err != nil {
		return models.Expense{}, err
	}
	return models.Expense{
		ExpenseDate:   date,
		Category:      r.Category,
		Amount:        r.Amount,
		PaymentMethod: strings.TrimSpace(r.PaymentMethod),
		Note:          strings.TrimSpace(r.Note),
	}, nil
}
