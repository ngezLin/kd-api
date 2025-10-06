package models

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
    ID          uint              `gorm:"primaryKey" json:"id"`
    Status      string            `gorm:"type:enum('draft','completed','refunded');default:'draft'" json:"status"`
    Total       float64           `gorm:"not null;default:0" json:"total"`
    Payment     *float64          `json:"payment,omitempty"`
    Change      *float64          `json:"change,omitempty"`
    PaymentType *string           `gorm:"type:enum('cash','qris','debit','credit')" json:"payment_type,omitempty"`
    Items       []TransactionItem `json:"items"`
    Note        *string           `gorm:"type:text" json:"note,omitempty"`

    CreatedAt   time.Time         `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt   gorm.DeletedAt    `gorm:"index" json:"-"`
}
