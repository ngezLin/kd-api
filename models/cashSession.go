package models

import "time"

type CashSession struct {
	ID uint `gorm:"primaryKey"`

	UserID uint `gorm:"not null"`

	OpeningCash float64 `gorm:"not null"`

	TotalCashIn     float64 `gorm:"default:0"`
	TotalChange     float64 `gorm:"default:0"`
	TotalRefundCash float64 `gorm:"default:0"`

	ExpectedCash float64  `gorm:"default:0"`
	ClosingCash  *float64
	Difference   *float64

	Status string `gorm:"type:enum('open','closed');default:'open'"`

	OpenedAt time.Time
	ClosedAt *time.Time
}
