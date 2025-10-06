package models

import (
	"time"

	"gorm.io/gorm"
)

type Item struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"unique;type:varchar(100);not null" json:"name"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	Stock       int            `gorm:"not null;default:0" json:"stock"`
	BuyPrice    float64        `gorm:"not null" json:"buy_price"`
	Price       float64        `gorm:"not null" json:"price"`
	ImageURL    *string        `gorm:"type:varchar(255)" json:"image_url,omitempty" nullable:"true"`
    CreatedAt   time.Time         `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt   gorm.DeletedAt    `gorm:"index" json:"-"`
}
