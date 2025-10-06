package models

type TransactionItem struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	TransactionID uint    `gorm:"not null" json:"transaction_id"`
	ItemID        uint    `gorm:"not null" json:"item_id"`
	Quantity      int     `gorm:"not null;default:1" json:"quantity"`
	Price         float64 `gorm:"not null" json:"price"`
	Subtotal      float64 `gorm:"not null" json:"subtotal"`

	// Relasi
	Item Item `gorm:"foreignKey:ItemID" json:"item"`
}