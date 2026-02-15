package services

import (
	"fmt"
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"

	"gorm.io/gorm"
)

type InventoryService interface {
	LogStockChange(tx *gorm.DB, itemID uint, change int, logType string, refID string, userID *uint, note string) error
	GetInventoryHistory(filter dtos.InventoryFilter) (*dtos.InventoryListResponse, error)
}

type inventoryService struct{}

func NewInventoryService() InventoryService {
	return &inventoryService{}
}

func (s *inventoryService) GetInventoryHistory(filter dtos.InventoryFilter) (*dtos.InventoryListResponse, error) {
	var logs []models.InventoryLog
	var total int64

	db := config.DB.Model(&models.InventoryLog{}).Where("item_id = ?", filter.ItemID)

	if filter.StartDate != "" {
		db = db.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		// assuming end date is inclusive for the day, might need time parsing but string compare works for YYYY-MM-DD if time is 00:00:00 usually. 
		// Ideally pass +1 day logic. Using string compare for now like other services.
		// Actually transactionService used time.Parse. Let's stick to simple string if possible or replicate standard pattern.
		// TransactionService used:
		// start, _ := time.Parse("2006-01-02", filter.Date)
		// end := start.Add(24 * time.Hour)
		// Here we allow ranges.
		db = db.Where("created_at <= ?", filter.EndDate+" 23:59:59")
	}

	if filter.Type != "" {
		db = db.Where("type = ?", filter.Type)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	if err := db.Preload("User").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return &dtos.InventoryListResponse{
		Data:       logs,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      total,
		TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
	}, nil
}

func (s *inventoryService) LogStockChange(tx *gorm.DB, itemID uint, change int, logType string, refID string, userID *uint, note string) error {
	// 1. Get current stock to ensure accuracy (locking row would be ideal but simple read is start)
	var item models.Item
	if err := tx.First(&item, itemID).Error; err != nil {
		return fmt.Errorf("item not found for inventory log: %w", err)
	}

	// 2. Create Log
	log := models.InventoryLog{
		ItemID:      itemID,
		Change:      change,
		FinalStock:  item.Stock, // This assumes the item.Stock has already been updated in the DB by the caller
		Type:        logType,
		ReferenceID: refID,
		UserID:      userID,
		Note:        note,
	}

	if err := tx.Create(&log).Error; err != nil {
		return fmt.Errorf("failed to create inventory log: %w", err)
	}

	return nil
}
