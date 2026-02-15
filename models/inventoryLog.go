package models

import (
	"time"
)

type InventoryLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ItemID      uint      `gorm:"not null;index" json:"item_id"`
	Change      int       `gorm:"not null" json:"change"`       // Positive for IN, Negative for OUT
	FinalStock  int       `gorm:"not null" json:"final_stock"`  // Stock after change
	Type        string    `gorm:"type:enum('sale','refund','adjustment','restock','audit','delete');not null" json:"type"`
	ReferenceID string    `gorm:"type:varchar(50)" json:"reference_id,omitempty"` // e.g., "TX-1001"
	Note        string    `gorm:"type:text" json:"note,omitempty"`
	UserID      *uint     `gorm:"index" json:"user_id,omitempty"` // Who caused the change
	CreatedAt   time.Time `gorm:"autoCreateTime;index" json:"created_at"`

	// Relations
	Item Item  `gorm:"foreignKey:ItemID" json:"item"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
