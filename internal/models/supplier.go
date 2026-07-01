package models

import "time"

type Supplier struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	Note      string    `json:"note"`
	VisitDays []string  `json:"visit_days" gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
