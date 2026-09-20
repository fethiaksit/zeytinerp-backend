package models

import "time"

type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string     `json:"-" gorm:"not null"`
	FirstName    string     `json:"first_name" gorm:"default:''"`
	LastName     string     `json:"last_name" gorm:"default:''"`
	Name         string     `json:"name" gorm:"default:''"`
	Phone        *string    `json:"phone"`
	Role         string     `json:"role" gorm:"not null;default:'cashier'"`
	IsActive     bool       `json:"is_active" gorm:"not null;default:true"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (u User) FullName() string {
	if u.FirstName != "" || u.LastName != "" {
		if u.FirstName != "" && u.LastName != "" {
			return u.FirstName + " " + u.LastName
		}
		if u.FirstName != "" {
			return u.FirstName
		}
		return u.LastName
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}
