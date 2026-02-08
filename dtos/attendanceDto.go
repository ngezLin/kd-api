package dtos

import (
	"kd-api/models"
	"time"
)

type CreateAttendanceInput struct {
	UserID uint   `json:"user_id" binding:"required"`
	Status string `json:"status" binding:"required,oneof=present absent off"`
	Note   string `json:"note"`
}

type AttendanceResponse struct {
	ID        uint           `json:"id"`
	UserID    uint           `json:"user_id"`
	Date      time.Time      `json:"date"`
	Status    string         `json:"status"`
	Note      *string        `json:"note"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	User      models.User    `json:"user"`
}
