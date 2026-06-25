package models

import "time"

type Customer struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"not null"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	CustomerType string    `json:"customer_type" gorm:"not null;default:normal"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
