package dtos

import (
	"kd-api/models"
)

type OpenCashSessionInput struct {
	OpeningCash float64 `json:"opening_cash" binding:"required"`
}

type CloseCashSessionInput struct {
	ClosingCash float64 `json:"closing_cash" binding:"required"`
}

type CashSessionHistoryFilter struct {
	Page      int
	PageSize  int
	StartDate string
	EndDate   string
}

type CashSessionListResponse struct {
	Data       []models.CashSession `json:"data"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}
