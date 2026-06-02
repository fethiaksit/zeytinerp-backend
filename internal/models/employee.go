package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Employee struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Name      string          `json:"name" gorm:"not null"`
	Phone     string          `json:"phone"`
	DailyWage decimal.Decimal `json:"daily_wage" gorm:"type:numeric(12,2);not null"`
	IsActive  bool            `json:"is_active" gorm:"not null;default:true"`
	Note      string          `json:"note"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
