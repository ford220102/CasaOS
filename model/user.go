package model

import (
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uint   `json:"id" gorm:"primaryKey"`
	Username string `json:"username" gorm:"unique;not null"`
	Password string `json:"-" gorm:"not null"` // hash hasła
	Email    string `json:"email" gorm:"unique"`
	Role     string `json:"role" gorm:"default:'user'"` // 'user' lub 'admin'
	Enabled  bool   `json:"enabled" gorm:"default:true"`
}

// Claims struktura dla JWT
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Result standardowa odpowiedź API
type Result struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
