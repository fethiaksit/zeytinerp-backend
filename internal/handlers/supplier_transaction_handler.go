package handlers

import (
	"context"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

type SupplierTransactionHandler struct{ DB *gorm.DB }

type supplierTransactionRequest struct {
	SupplierID      uint             `json:"supplier_id"`
	TransactionDate string           `json:"transaction_date"`
	Type            string           `json:"type"`
	Amount          decimal.Decimal  `json:"amount"`
	AmountOriginal  decimal.Decimal  `json:"amount_original"`
	Currency        string           `json:"currency"`
	ExchangeRate    *decimal.Decimal `json:"exchange_rate"`
	PaymentMethod   string           `json:"payment_method"`
	InvoiceNo       string           `json:"invoice_no"`
	Note            string           `json:"note"`
}

func NewSupplierTransactionHandler(db *gorm.DB) *SupplierTransactionHandler {
	return &SupplierTransactionHandler{DB: db}
}

func (h *SupplierTransactionHandler) Create(c *gin.Context) {
	req, files, err := bindSupplierTransactionRequest(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	tx, err := req.toModel(ctx, h.DB)
	if err != nil {
		log.Printf("[SupplierTransaction Error] request validation failed: %v", err)
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(files) > 0 && !supplierTransactionAllowsFiles(tx.Type) {
		fail(c, http.StatusBadRequest, supplierTransactionFileTypeError())
		return
	}
	log.Printf("[SupplierTransaction] currency=%s amount_original=%s exchange_rate=%s amount_try=%s", tx.Currency, tx.AmountOriginal.String(), tx.ExchangeRate.String(), tx.AmountTRY.String())
	if len(files) == 0 {
		if err := h.DB.Create(&tx).Error; err != nil {
			log.Printf("[SupplierTransaction Error] database insert failed: %v", err)
			handleDBError(c, err)
			return
		}
	} else {
		if err := h.DB.Transaction(func(dbtx *gorm.DB) error {
			if err := dbtx.Create(&tx).Error; err != nil {
				return err
			}
			records, err := h.saveSupplierTransactionFiles(c, dbtx, tx.ID, files)
			if err != nil {
				return err
			}
			tx.Files = records
			if err := syncSupplierTransactionPrimaryFile(dbtx, &tx); err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Printf("[SupplierTransaction Error] create with files failed: %v", err)
			handleSupplierTransactionFileError(c, err)
			return
		}
	}
	log.Printf("[SupplierTransaction] database record created id=%d currency=%s amount_original=%s exchange_rate=%s amount_try=%s", tx.ID, tx.Currency, tx.AmountOriginal.String(), tx.ExchangeRate.String(), tx.AmountTRY.String())
	created(c, tx)
}

func (h *SupplierTransactionHandler) List(c *gin.Context) {
	var txs []models.SupplierTransaction
	query := h.DB.Preload("Files", func(db *gorm.DB) *gorm.DB {
		return db.Order("page_order asc, id asc")
	}).Order("transaction_date desc, id desc")
	if supplierID := c.Query("supplier_id"); supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if txType := strings.TrimSpace(c.Query("type")); txType != "" {
		query = query.Where("type = ?", txType)
	}
	if err := query.Find(&txs).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, txs)
}

func (h *SupplierTransactionHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var tx models.SupplierTransaction
	if err := h.DB.Preload("Files", func(db *gorm.DB) *gorm.DB {
		return db.Order("page_order asc, id asc")
	}).First(&tx, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, tx)
}

func (h *SupplierTransactionHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	req, files, err := bindSupplierTransactionRequest(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	replaceFiles, deleteAllFiles, deleteFileIDs, err := supplierTransactionFileUpdateOptions(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	updatedTx, err := req.toModel(ctx, h.DB)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if (len(files) > 0 || replaceFiles || deleteAllFiles || len(deleteFileIDs) > 0) && !supplierTransactionAllowsFiles(updatedTx.Type) {
		fail(c, http.StatusBadRequest, supplierTransactionFileTypeError())
		return
	}

	var existing models.SupplierTransaction
	if err := h.DB.First(&existing, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	updatedTx.ID = existing.ID
	updatedTx.CreatedAt = existing.CreatedAt
	updatedTx.ImageURL = existing.ImageURL
	updatedTx.FilePath = existing.FilePath
	if !supplierTransactionUsesPrimaryFile(updatedTx.Type) {
		updatedTx.ImageURL = ""
		updatedTx.FilePath = ""
	}

	pathsToRemove := make([]string, 0)
	if err := h.DB.Transaction(func(dbtx *gorm.DB) error {
		if err := dbtx.Save(&updatedTx).Error; err != nil {
			return err
		}

		if replaceFiles || deleteAllFiles {
			filesToDelete, err := supplierTransactionFilesForDelete(dbtx, id, nil)
			if err != nil {
				return err
			}
			if err := deleteSupplierTransactionFileRecords(dbtx, filesToDelete); err != nil {
				return err
			}
			pathsToRemove = append(pathsToRemove, supplierTransactionFilePaths(filesToDelete)...)
		} else if len(deleteFileIDs) > 0 {
			filesToDelete, err := supplierTransactionFilesForDelete(dbtx, id, deleteFileIDs)
			if err != nil {
				return err
			}
			if len(filesToDelete) != len(deleteFileIDs) {
				return newSupplierTransactionFileError(http.StatusBadRequest, "delete_file_ids contains invalid file id")
			}
			if err := deleteSupplierTransactionFileRecords(dbtx, filesToDelete); err != nil {
				return err
			}
			pathsToRemove = append(pathsToRemove, supplierTransactionFilePaths(filesToDelete)...)
		}

		if len(files) > 0 {
			if _, err := h.saveSupplierTransactionFiles(c, dbtx, id, files); err != nil {
				return err
			}
		}
		if len(files) > 0 || replaceFiles || deleteAllFiles || len(deleteFileIDs) > 0 || updatedTx.Type != existing.Type {
			if err := syncSupplierTransactionPrimaryFile(dbtx, &updatedTx); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		handleSupplierTransactionFileError(c, err)
		return
	}
	removeSupplierTransactionFilePaths(pathsToRemove)

	var tx models.SupplierTransaction
	if err := h.DB.Preload("Files", func(db *gorm.DB) *gorm.DB {
		return db.Order("page_order asc, id asc")
	}).First(&tx, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, tx)
}

func (h *SupplierTransactionHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var tx models.SupplierTransaction
	if err := h.DB.Preload("Files").First(&tx, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	filePaths := supplierTransactionFilePaths(tx.Files)
	if strings.TrimSpace(tx.FilePath) != "" {
		filePaths = append(filePaths, tx.FilePath)
	}
	if err := h.DB.Delete(&models.SupplierTransaction{}, id).Error; err != nil {
		handleDBError(c, err)
		return
	}
	removeSupplierTransactionFilePaths(filePaths)
	ok(c, gin.H{"deleted": true})
}

func (h *SupplierTransactionHandler) DeleteInvoice(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}

	var invoice models.SupplierTransaction
	if err := h.DB.Preload("Files").Where("id = ? AND type = ?", id, "invoice").First(&invoice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Fatura bulunamadı")
			return
		}
		handleDBError(c, err)
		return
	}

	filePaths := make([]string, 0, len(invoice.Files))
	for _, file := range invoice.Files {
		if strings.TrimSpace(file.FilePath) != "" {
			filePaths = append(filePaths, file.FilePath)
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Delete(&models.SupplierTransaction{}, invoice.ID).Error
	}); err != nil {
		log.Printf("[Invoice Delete Error] database delete failed for invoice %d: %v", invoice.ID, err)
		handleDBError(c, err)
		return
	}

	for _, filePath := range filePaths {
		if err := os.Remove(filePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Printf("[Invoice Delete] file already missing: %s", filePath)
			} else {
				log.Printf("[Invoice Delete Error] file delete failed for %s: %v", filePath, err)
			}
			continue
		}
		log.Printf("[Invoice Delete] file deleted: %s", filePath)
	}

	log.Printf("[Invoice Delete] invoice deleted: id=%d files=%d", invoice.ID, len(filePaths))
	c.JSON(http.StatusOK, Response{Success: true, Message: "Fatura silindi"})
}

func (r supplierTransactionRequest) toModel(ctx context.Context, db *gorm.DB) (models.SupplierTransaction, error) {
	if r.SupplierID == 0 {
		return models.SupplierTransaction{}, errRequired("supplier_id")
	}
	r.Type = strings.TrimSpace(r.Type)
	if !validateType(r.Type, map[string]bool{
		"invoice":  true,
		"purchase": true,
		"payment":  true,
		"return":   true,
	}) {
		return models.SupplierTransaction{}, errInvalidType("type")
	}
	originalAmount := r.AmountOriginal
	if originalAmount.IsZero() {
		originalAmount = r.Amount
	}
	if err := positiveDecimal(originalAmount, "amount"); err != nil {
		return models.SupplierTransaction{}, err
	}
	currency, err := services.NormalizeCurrency(r.Currency)
	if err != nil {
		return models.SupplierTransaction{}, err
	}
	exchangeRate := decimal.NewFromInt(1)
	if currency != "TRY" {
		if r.ExchangeRate != nil {
			if err := positiveDecimal(*r.ExchangeRate, "exchange_rate"); err != nil {
				return models.SupplierTransaction{}, err
			}
			exchangeRate = *r.ExchangeRate
		} else {
			latestRate, err := services.LatestRateToTRY(ctx, db, currency)
			if err != nil {
				return models.SupplierTransaction{}, err
			}
			exchangeRate = decimal.NewFromFloat(latestRate.RateToTRY)
		}
	}
	amountTRY := originalAmount.Mul(exchangeRate).Round(2)
	paymentMethod, err := normalizeSupplierPaymentMethod(r.PaymentMethod, r.Type)
	if err != nil {
		return models.SupplierTransaction{}, err
	}
	date, err := parseDate(r.TransactionDate)
	if err != nil {
		return models.SupplierTransaction{}, err
	}
	return models.SupplierTransaction{
		SupplierID:      r.SupplierID,
		TransactionDate: date,
		Type:            r.Type,
		Amount:          originalAmount,
		Currency:        currency,
		ExchangeRate:    exchangeRate,
		AmountOriginal:  originalAmount,
		AmountTRY:       amountTRY,
		PaymentMethod:   paymentMethod,
		InvoiceNo:       strings.TrimSpace(r.InvoiceNo),
		Note:            strings.TrimSpace(r.Note),
	}, nil
}

func normalizeSupplierPaymentMethod(method, txType string) (string, error) {
	method = strings.TrimSpace(method)
	if method == "bank" {
		method = "bank_transfer"
	}
	if txType == "payment" && method == "" {
		return "", errRequired("payment_method")
	}
	if method == "" {
		return "", nil
	}
	if !validateType(method, map[string]bool{
		"cash":            true,
		"credit_card":     true,
		"current_account": true,
		"bank_transfer":   true,
		"other":           true,
	}) {
		return "", errInvalidType("payment_method")
	}
	return method, nil
}

func bindSupplierTransactionRequest(c *gin.Context) (supplierTransactionRequest, []*multipart.FileHeader, error) {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		var req supplierTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return supplierTransactionRequest{}, nil, errors.New("invalid json body")
		}
		return req, nil, nil
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInvoiceUploadSize)
	form, err := c.MultipartForm()
	if err != nil {
		return supplierTransactionRequest{}, nil, errors.New("invalid multipart form")
	}

	supplierID, err := parseUintFormValue(c.PostForm("supplier_id"), "supplier_id")
	if err != nil {
		return supplierTransactionRequest{}, nil, err
	}
	amount, err := parseDecimalFormValue(c.PostForm("amount"), "amount")
	if err != nil {
		return supplierTransactionRequest{}, nil, err
	}
	amountOriginal, err := parseDecimalFormValue(c.PostForm("amount_original"), "amount_original")
	if err != nil {
		return supplierTransactionRequest{}, nil, err
	}
	exchangeRate, err := parseOptionalDecimalFormValue(c.PostForm("exchange_rate"), "exchange_rate")
	if err != nil {
		return supplierTransactionRequest{}, nil, err
	}

	files := supplierTransactionUploadFilesFromForm(form.File)
	req := supplierTransactionRequest{
		SupplierID:      supplierID,
		TransactionDate: c.PostForm("transaction_date"),
		Type:            c.PostForm("type"),
		Amount:          amount,
		AmountOriginal:  amountOriginal,
		Currency:        c.PostForm("currency"),
		ExchangeRate:    exchangeRate,
		PaymentMethod:   c.PostForm("payment_method"),
		InvoiceNo:       c.PostForm("invoice_no"),
		Note:            c.PostForm("note"),
	}
	return req, files, nil
}

func parseUintFormValue(value, field string) (uint, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, errInvalidType(field)
	}
	return uint(parsed), nil
}

func parseDecimalFormValue(value, field string) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return decimal.Zero, nil
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, errInvalidType(field)
	}
	return parsed, nil
}

func parseOptionalDecimalFormValue(value, field string) (*decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return nil, errInvalidType(field)
	}
	return &parsed, nil
}

func supplierTransactionFileUpdateOptions(c *gin.Context) (bool, bool, []uint, error) {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return false, false, nil, nil
	}
	replaceFiles := parseBoolFormValue(c.PostForm("replace_files"))
	deleteAllFiles := parseBoolFormValue(c.PostForm("delete_files"))
	deleteFileIDs, err := parseUintListFormValues(append(c.PostFormArray("delete_file_ids"), c.PostFormArray("delete_file_id")...))
	if err != nil {
		return false, false, nil, err
	}
	return replaceFiles, deleteAllFiles, deleteFileIDs, nil
}

func parseBoolFormValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseUintListFormValues(values []string) ([]uint, error) {
	ids := make([]uint, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseUint(part, 10, 64)
			if err != nil || id == 0 {
				return nil, errInvalidType("delete_file_ids")
			}
			ids = append(ids, uint(id))
		}
	}
	return ids, nil
}
