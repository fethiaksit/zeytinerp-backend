package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

func TestSupplierCashPaymentsBetween(t *testing.T) {
	db := newSupplierPaymentTestDB(t)

	supplier := models.Supplier{Name: "Test Supplier"}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	date := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)
	transactions := []models.SupplierTransaction{
		newSupplierPayment(supplier.ID, date, "cash", "100"),
		newSupplierPayment(supplier.ID, date, "nakit", "25"),
		newSupplierPayment(supplier.ID, date, "bank_transfer", "300"),
		newSupplierPayment(supplier.ID, date, "credit_card", "400"),
		newSupplierPayment(supplier.ID, date.AddDate(0, 0, -1), "cash", "500"),
		newSupplierPayment(supplier.ID, date.AddDate(0, 0, 1), "cash", "600"),
		newSupplierTransaction(supplier.ID, date, "invoice", "cash", "700"),
	}
	if err := db.Create(&transactions).Error; err != nil {
		t.Fatalf("create transactions: %v", err)
	}

	total, err := SupplierCashPaymentsBetween(db, date, date.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("calculate cash payments: %v", err)
	}
	if !total.Equal(decimal.RequireFromString("125")) {
		t.Fatalf("cash payment total = %s, want 125", total)
	}
}

func TestDailyCashOutflowBetweenAddsEachSourceOnce(t *testing.T) {
	db := newSupplierPaymentTestDB(t)
	supplier := models.Supplier{Name: "Test Supplier"}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	date := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.Expense{ExpenseDate: date, Category: "diger", Amount: decimal.RequireFromString("40")}).Error; err != nil {
		t.Fatalf("create expense: %v", err)
	}
	transactions := []models.SupplierTransaction{
		newSupplierPayment(supplier.ID, date, "cash", "60"),
		newSupplierPayment(supplier.ID, date, "bank_transfer", "900"),
	}
	if err := db.Create(&transactions).Error; err != nil {
		t.Fatalf("create transactions: %v", err)
	}

	total, supplierCash, err := DailyCashOutflowBetween(db, date, date.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("calculate daily cash outflow: %v", err)
	}
	if !supplierCash.Equal(decimal.RequireFromString("60")) {
		t.Fatalf("supplier cash total = %s, want 60", supplierCash)
	}
	if !total.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("daily cash outflow = %s, want 100", total)
	}
}

func newSupplierPaymentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.Supplier{}, &models.SupplierTransaction{}, &models.Expense{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func newSupplierPayment(supplierID uint, date time.Time, method, amount string) models.SupplierTransaction {
	return newSupplierTransaction(supplierID, date, "payment", method, amount)
}

func newSupplierTransaction(supplierID uint, date time.Time, txType, method, amount string) models.SupplierTransaction {
	value := decimal.RequireFromString(amount)
	return models.SupplierTransaction{
		SupplierID:      supplierID,
		TransactionDate: date,
		Type:            txType,
		Amount:          value,
		Currency:        "TRY",
		ExchangeRate:    decimal.NewFromInt(1),
		AmountOriginal:  value,
		AmountTRY:       value,
		PaymentMethod:   method,
	}
}
