package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

const (
	invoiceUploadDir     = "uploads/invoices"
	maxInvoiceFileSize   = 10 << 20
	maxInvoiceUploadSize = 120 << 20
)

var allowedInvoiceMIMEs = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

var allowedInvoiceExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".pdf":  true,
}

func (h *SupplierTransactionHandler) UploadFiles(c *gin.Context) {
	log.Println("UPLOAD START")
	transactionID, valid := parseID(c)
	if !valid {
		log.Println("UPLOAD ERROR: invalid transaction id")
		return
	}
	log.Printf("TRANSACTION ID: %d", transactionID)

	var tx models.SupplierTransaction
	if err := h.DB.First(&tx, transactionID).Error; err != nil {
		log.Printf("UPLOAD ERROR: transaction lookup failed: %v", err)
		handleDBError(c, err)
		return
	}
	if tx.Type != "invoice" {
		log.Printf("UPLOAD ERROR: transaction %d type is %s", transactionID, tx.Type)
		fail(c, http.StatusBadRequest, "Fatura dosyası sadece Gelen Fatura hareketine eklenebilir")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInvoiceUploadSize)
	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("UPLOAD ERROR: multipart parse failed: %v", err)
		fail(c, http.StatusBadRequest, "invalid multipart form")
		return
	}
	files := form.File["files"]
	log.Printf("FILES COUNT: %d", len(files))
	if len(files) == 0 {
		log.Println("UPLOAD ERROR: files field is empty")
		fail(c, http.StatusBadRequest, errRequired("files").Error())
		return
	}
	uploadDir, err := invoiceUploadDirPath()
	if err != nil {
		log.Printf("UPLOAD ERROR: upload path failed: %v", err)
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("UPLOAD ERROR: mkdir failed: %v", err)
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	var maxPageOrder int
	if err := h.DB.Model(&models.SupplierTransactionFile{}).
		Where("supplier_transaction_id = ?", transactionID).
		Select("COALESCE(MAX(page_order), 0)").
		Scan(&maxPageOrder).Error; err != nil {
		log.Printf("UPLOAD ERROR: max page query failed: %v", err)
		handleDBError(c, err)
		return
	}

	records := make([]models.SupplierTransactionFile, 0, len(files))
	savedPaths := make([]string, 0, len(files))
	for i, fileHeader := range files {
		mimeType, ext, err := validateInvoiceUpload(fileHeader)
		if err != nil {
			log.Printf("UPLOAD ERROR: validation failed for %s: %v", fileHeader.Filename, err)
			cleanupFiles(savedPaths)
			fail(c, http.StatusBadRequest, err.Error())
			return
		}

		storedName := newUUID() + ext
		filePath := filepath.Join(uploadDir, storedName)
		if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
			log.Printf("UPLOAD ERROR: save failed for %s: %v", fileHeader.Filename, err)
			cleanupFiles(savedPaths)
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("SAVED FILE: %s", filePath)
		savedPaths = append(savedPaths, filePath)

		records = append(records, models.SupplierTransactionFile{
			SupplierTransactionID: transactionID,
			FileName:              strings.TrimSpace(fileHeader.Filename),
			FilePath:              filePath,
			FileURL:               "/uploads/invoices/" + storedName,
			MimeType:              mimeType,
			Size:                  fileHeader.Size,
			PageOrder:             maxPageOrder + i + 1,
		})
	}

	if err := h.DB.Create(&records).Error; err != nil {
		log.Printf("UPLOAD ERROR: db create failed: %v", err)
		cleanupFiles(savedPaths)
		handleDBError(c, err)
		return
	}
	for _, record := range records {
		log.Printf("DB RECORD CREATED: supplier_transaction_file_id=%d file_url=%s", record.ID, record.FileURL)
	}
	created(c, records)
}

func (h *SupplierTransactionHandler) ListFiles(c *gin.Context) {
	transactionID, valid := parseID(c)
	if !valid {
		return
	}
	var tx models.SupplierTransaction
	if err := h.DB.First(&tx, transactionID).Error; err != nil {
		handleDBError(c, err)
		return
	}

	var files []models.SupplierTransactionFile
	if err := h.DB.Where("supplier_transaction_id = ?", transactionID).
		Order("page_order asc, id asc").
		Find(&files).Error; err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, files)
}

func (h *SupplierTransactionHandler) DeleteFile(c *gin.Context) {
	fileID, valid := parseParamUint(c, "file_id")
	if !valid {
		return
	}

	var file models.SupplierTransactionFile
	if err := h.DB.First(&file, fileID).Error; err != nil {
		handleDBError(c, err)
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.SupplierTransactionFile{}, fileID).Error; err != nil {
			return err
		}
		if err := os.Remove(file.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

func ServeInvoiceFile(c *gin.Context) {
	requested := strings.TrimPrefix(c.Param("filepath"), "/")
	if requested == "" || strings.Contains(requested, "..") || strings.ContainsAny(requested, `/\`) {
		fail(c, http.StatusBadRequest, "invalid file path")
		return
	}

	uploadDir, err := invoiceUploadDirPath()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	fullPath := filepath.Join(uploadDir, requested)
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fail(c, http.StatusNotFound, "file not found")
			return
		}
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.File(fullPath)
}

func invoiceUploadDirPath() (string, error) {
	return filepath.Abs(invoiceUploadDir)
}

func validateInvoiceUpload(fileHeader *multipart.FileHeader) (string, string, error) {
	if fileHeader.Size <= 0 {
		return "", "", errors.New("file is empty")
	}
	if fileHeader.Size > maxInvoiceFileSize {
		return "", "", errors.New("file size cannot exceed 10MB")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedInvoiceExts[ext] {
		return "", "", errors.New("file extension is not allowed")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}

	mimeType := detectInvoiceMIME(buffer[:n])
	if !allowedInvoiceMIMEs[mimeType] {
		return "", "", errors.New("file mime type is not allowed")
	}
	if !extensionMatchesMIME(ext, mimeType) {
		return "", "", errors.New("file extension does not match mime type")
	}

	return mimeType, ext, nil
}

func detectInvoiceMIME(buffer []byte) string {
	if len(buffer) >= 12 && string(buffer[0:4]) == "RIFF" && string(buffer[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(buffer)
}

func extensionMatchesMIME(ext, mimeType string) bool {
	switch ext {
	case ".jpg", ".jpeg":
		return mimeType == "image/jpeg"
	case ".png":
		return mimeType == "image/png"
	case ".webp":
		return mimeType == "image/webp"
	case ".pdf":
		return mimeType == "application/pdf"
	default:
		return false
	}
}

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strings.ReplaceAll(fmt.Sprintf("%d", os.Getpid()), "-", "") + "-" + hex.EncodeToString([]byte(fmt.Sprintf("%p", &b)))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func cleanupFiles(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func parseParamUint(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return uint(id), true
}
