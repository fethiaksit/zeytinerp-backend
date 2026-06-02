package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type CashReportHandler struct{ DB *gorm.DB }

type cashReportRequest struct {
	ReportDate       string          `json:"report_date"`
	CashAmount       decimal.Decimal `json:"cash_amount"`
	PosAmount        decimal.Decimal `json:"pos_amount"`
	QRAmount         decimal.Decimal `json:"qr_amount"`
	CreditCollection decimal.Decimal `json:"credit_collection"`
	CreditGiven      decimal.Decimal `json:"credit_given"`
	Note             string          `json:"note"`
}

func NewCashReportHandler(db *gorm.DB) *CashReportHandler { return &CashReportHandler{DB: db} }

func (h *CashReportHandler) Create(c *gin.Context) {
	var req cashReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	report, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.DB.Create(&report).Error; err != nil {
		handleDBError(c, err)
		return
	}
	created(c, report)
}

func (h *CashReportHandler) List(c *gin.Context) {
	var reports []models.CashReport
	if err := h.DB.Order("report_date desc, id desc").Find(&reports).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, reports)
}

func (h *CashReportHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var report models.CashReport
	if err := h.DB.First(&report, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, report)
}

func (h *CashReportHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req cashReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	report, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var existing models.CashReport
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	report.ID = existing.ID
	report.CreatedAt = existing.CreatedAt
	if err := h.DB.Save(&report).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, report)
}

func (h *CashReportHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.DB.Delete(&models.CashReport{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r cashReportRequest) toModel() (models.CashReport, error) {
	date, err := parseDate(r.ReportDate)
	if err != nil {
		return models.CashReport{}, err
	}
	for field, value := range map[string]decimal.Decimal{
		"cash_amount":       r.CashAmount,
		"pos_amount":        r.PosAmount,
		"qr_amount":         r.QRAmount,
		"credit_collection": r.CreditCollection,
		"credit_given":      r.CreditGiven,
	} {
		if err := notNegativeDecimal(value, field); err != nil {
			return models.CashReport{}, err
		}
	}
	return models.CashReport{
		ReportDate:       date,
		CashAmount:       r.CashAmount,
		PosAmount:        r.PosAmount,
		QRAmount:         r.QRAmount,
		CreditCollection: r.CreditCollection,
		CreditGiven:      r.CreditGiven,
		Note:             r.Note,
	}, nil
}
