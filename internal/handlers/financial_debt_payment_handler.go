package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type FinancialDebtPaymentHandler struct{ DB *gorm.DB }

type financialDebtPaymentRequest struct {
	ID                   uint            `json:"id"`
	FinancialDebtID      uint            `json:"financial_debt_id"`
	FinancialDebtIDCamel uint            `json:"financialDebtId"`
	DebtID               uint            `json:"debt_id"`
	InstallmentID        *uint           `json:"installment_id"`
	PaymentDate          string          `json:"payment_date"`
	Amount               decimal.Decimal `json:"amount"`
	PaymentMethod        string          `json:"payment_method"`
	Note                 string          `json:"note"`
}

func NewFinancialDebtPaymentHandler(db *gorm.DB) *FinancialDebtPaymentHandler {
	return &FinancialDebtPaymentHandler{DB: db}
}

func (h *FinancialDebtPaymentHandler) Create(c *gin.Context) {
	var req financialDebtPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid json body")
		return
	}
	log.Printf("PAYMENT REQUEST RECEIVED: %+v", req)
	payment, err := req.toModel()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var debt models.FinancialDebt
	if err := h.DB.First(&debt, payment.FinancialDebtID).Error; err != nil {
		handleDBError(c, err)
		return
	}
	if payment.InstallmentID != nil {
		var installment models.FinancialDebtInstallment
		if err := h.DB.First(&installment, *payment.InstallmentID).Error; err != nil {
			handleDBError(c, err)
			return
		}
		if installment.FinancialDebtID != payment.FinancialDebtID {
			fail(c, http.StatusBadRequest, "installment_id does not belong to financial_debt_id")
			return
		}
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		if payment.InstallmentID == nil {
			return nil
		}
		return services.RecalculateFinancialInstallment(tx, *payment.InstallmentID)
	})
	if err != nil {
		handleDBError(c, err)
		return
	}
	created(c, payment)
}

func (h *FinancialDebtPaymentHandler) List(c *gin.Context) {
	var payments []models.FinancialDebtPayment
	query := h.DB.Order("payment_date desc, id desc")
	if debtID := c.Query("debt_id"); debtID != "" {
		query = query.Where("financial_debt_id = ?", debtID)
	}
	if installmentID := c.Query("installment_id"); installmentID != "" {
		query = query.Where("installment_id = ?", installmentID)
	}
	if err := query.Find(&payments).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, payments)
}

func (h *FinancialDebtPaymentHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var payment models.FinancialDebtPayment
	if err := h.DB.First(&payment, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.FinancialDebtPayment{}, id).Error; err != nil {
			return err
		}
		if payment.InstallmentID == nil {
			return nil
		}
		return services.RecalculateFinancialInstallment(tx, *payment.InstallmentID)
	})
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func (r financialDebtPaymentRequest) toModel() (models.FinancialDebtPayment, error) {
	if r.FinancialDebtID == 0 && r.DebtID != 0 {
		r.FinancialDebtID = r.DebtID
	}
	if r.FinancialDebtID == 0 && r.FinancialDebtIDCamel != 0 {
		r.FinancialDebtID = r.FinancialDebtIDCamel
	}
	if r.FinancialDebtID == 0 && r.ID != 0 {
		r.FinancialDebtID = r.ID
	}
	if r.FinancialDebtID == 0 {
		return models.FinancialDebtPayment{}, errRequired("financial_debt_id")
	}
	r.PaymentMethod = strings.TrimSpace(r.PaymentMethod)
	if !validateType(r.PaymentMethod, map[string]bool{
		"cash":          true,
		"bank_transfer": true,
		"credit_card":   true,
		"other":         true,
	}) {
		return models.FinancialDebtPayment{}, errInvalidType("payment_method")
	}
	if err := positiveDecimal(r.Amount, "amount"); err != nil {
		return models.FinancialDebtPayment{}, errRequired("amount")
	}
	paymentDate, err := parseDate(r.PaymentDate)
	if err != nil {
		return models.FinancialDebtPayment{}, err
	}
	return models.FinancialDebtPayment{
		FinancialDebtID: r.FinancialDebtID,
		InstallmentID:   r.InstallmentID,
		PaymentDate:     paymentDate,
		Amount:          r.Amount,
		PaymentMethod:   r.PaymentMethod,
		Note:            strings.TrimSpace(r.Note),
	}, nil
}
