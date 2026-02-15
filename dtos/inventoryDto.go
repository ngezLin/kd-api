package dtos

import (
	"kd-api/models"
)

type InventoryFilter struct {
	ItemID    uint   `form:"item_id" binding:"required"`
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
	Type      string `form:"type"`       // specific type filter
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}

type InventoryListResponse struct {
	Data       []models.InventoryLog `json:"data"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	Total      int64                 `json:"total"`
	TotalPages int                   `json:"total_pages"`
}
