package dtos

import "kd-api/models"

type TransactionItemInput struct {
	ItemID      uint     `json:"item_id"`
	Quantity    int      `json:"quantity"`
	CustomPrice *float64 `json:"customPrice,omitempty"`
}

type CreateTransactionInput struct {
	Status          string                 `json:"status"`
	PaymentAmount   *float64               `json:"paymentAmount,omitempty"`
	PaymentType     *string                `json:"paymentType,omitempty"`
	Note            *string                `json:"note,omitempty"`
	TransactionType *string                `json:"transaction_type,omitempty"`
	Discount        *float64               `json:"discount,omitempty"`
	Items           []TransactionItemInput `json:"items"`
}

type UpdateTransactionInput struct {
	Status          string   `json:"status"`
	Note            *string  `json:"note,omitempty"`
	TransactionType *string  `json:"transaction_type,omitempty"`
	Discount        *float64 `json:"discount,omitempty"`
}

type TransactionFilter struct {
	Page      int
	Limit     int
	StartDate string
	Date      string
	Status    string
}



type TransactionListResponse struct {
	Data       []models.Transaction `json:"data"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"totalPages"`
}
