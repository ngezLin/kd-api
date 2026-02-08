package dtos

import (
	"kd-api/models"
)

type itemResponseData interface{}

type ItemListResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type CreateItemInput struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Stock       int     `json:"stock"`
	BuyPrice    float64 `json:"buy_price"`
	Price       float64 `json:"price" binding:"required"`
	ImageURL    *string `json:"image_url"`
}

type UpdateItemInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Stock       int     `json:"stock"`
	BuyPrice    float64 `json:"buy_price"`
	Price       float64 `json:"price"`
	ImageURL    *string `json:"image_url"`
}

type ItemFilter struct {
	Page     int
	PageSize int
	Name     string
}

type CSVExport struct {
	FileName string
	Content  []byte
}

type BulkCreateItemInput []models.Item
