package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"kd-api/utils/common"
	"kd-api/utils/log"
	"kd-api/utils/pagination"
	"kd-api/utils/response"
	"strings"

	"gorm.io/gorm"
)

type ItemService interface {
	GetItems(filter dtos.ItemFilter, role string) (*dtos.ItemListResponse, error)
	GetItemByID(id string, role string) (interface{}, error)
	CreateItem(input dtos.CreateItemInput, userID *uint, clientIP string, role string) (interface{}, error)
	UpdateItem(id string, input dtos.UpdateItemInput, userID *uint, clientIP string, role string) (interface{}, error)
	DeleteItem(id string, userID *uint, clientIP string) error
	BulkCreateItems(inputs dtos.BulkCreateItemInput, userID *uint, clientIP string, role string) (interface{}, error)
	ExportItems(role string) (*dtos.CSVExport, error)
}

type itemService struct{}

func NewItemService() ItemService {
	return &itemService{}
}

func (s *itemService) GetItems(filter dtos.ItemFilter, role string) (*dtos.ItemListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	p := pagination.New(filter.Page, filter.PageSize)

	var items []models.Item
	var total int64

	query := config.DB.Model(&models.Item{})

	if filter.Name != "" {
		for _, term := range strings.Fields(strings.ToLower(strings.TrimSpace(filter.Name))) {
			query = query.Where("LOWER(name) LIKE ?", "%"+term+"%")
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Offset(p.Offset).
		Limit(p.PageSize).
		Find(&items).Error; err != nil {
		return nil, err
	}

	meta := dtos.PaginationMeta{
		Page:       p.Page,
		Limit:      p.PageSize,
		Total:      total,
		TotalPages: int((total + int64(p.PageSize) - 1) / int64(p.PageSize)),
	}

	return &dtos.ItemListResponse{
		Data: response.FilterItemsForRole(items, role),
		Meta: meta,
	}, nil
}

func (s *itemService) GetItemByID(id string, role string) (interface{}, error) {
	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		return nil, errors.New("Item not found")
	}
	return response.FilterItemForRole(item, role), nil
}

func (s *itemService) CreateItem(input dtos.CreateItemInput, userID *uint, clientIP string, role string) (interface{}, error) {
	var existing models.Item
	if err := config.DB.Where("name = ?", input.Name).First(&existing).Error; err == nil {
		return nil, errors.New("Item dengan nama ini sudah ada")
	}

	item := models.Item{
		Name:        input.Name,
		Description: input.Description,
		Stock:       input.Stock,
		BuyPrice:    input.BuyPrice,
		Price:       input.Price,
		ImageURL:    input.ImageURL,
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}

		description := fmt.Sprintf("Item '%s' created", item.Name)
		return log.CreateItemAuditLog(
			tx,
			"create",
			item.ID,
			nil,
			&item,
			userID,
			clientIP,
			description,
		)
	})

	if err != nil {
		return nil, err
	}

	return response.FilterItemForRole(item, role), nil
}

func (s *itemService) UpdateItem(id string, input dtos.UpdateItemInput, userID *uint, clientIP string, role string) (interface{}, error) {
	var oldItem models.Item
	if err := config.DB.First(&oldItem, id).Error; err != nil {
		return nil, errors.New("Item not found")
	}

	var existing models.Item
	if err := config.DB.Where("name = ? AND id != ?", input.Name, oldItem.ID).
		First(&existing).Error; err == nil {
		return nil, errors.New("Item dengan nama ini sudah ada")
	}

	oldCopy := oldItem

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		oldItem.Name = input.Name
		oldItem.Description = input.Description
		oldItem.Stock = input.Stock
		oldItem.BuyPrice = input.BuyPrice
		oldItem.Price = input.Price
		oldItem.ImageURL = input.ImageURL

		if err := tx.Save(&oldItem).Error; err != nil {
			return err
		}

		description := fmt.Sprintf("Item '%s' updated", oldItem.Name)
		return log.CreateItemAuditLog(
			tx,
			"update",
			oldItem.ID,
			&oldCopy,
			&oldItem,
			userID,
			clientIP,
			description,
		)
	})

	if err != nil {
		return nil, err
	}

	return response.FilterItemForRole(oldItem, role), nil
}

func (s *itemService) DeleteItem(id string, userID *uint, clientIP string) error {
	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		return errors.New("Item not found")
	}

	itemCopy := item

	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}

		description := fmt.Sprintf("Item '%s' deleted", itemCopy.Name)
		return log.CreateItemAuditLog(
			tx,
			"delete",
			itemCopy.ID,
			&itemCopy,
			nil,
			userID,
			clientIP,
			description,
		)
	})
}

func (s *itemService) BulkCreateItems(inputs dtos.BulkCreateItemInput, userID *uint, clientIP string, role string) (interface{}, error) {
	items := []models.Item(inputs)

	for i := range items {
		if items[i].Description != nil && *items[i].Description == "" {
			items[i].Description = nil
		}
		if items[i].ImageURL != nil && *items[i].ImageURL == "" {
			items[i].ImageURL = nil
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&items).Error; err != nil {
			return err
		}

		for _, item := range items {
			description := fmt.Sprintf("Item '%s' created via bulk import", item.Name)
			if err := log.CreateItemAuditLog(
				tx,
				"create",
				item.ID,
				nil,
				&item,
				userID,
				clientIP,
				description,
			); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response.FilterItemsForRole(items, role), nil
}

func (s *itemService) ExportItems(role string) (*dtos.CSVExport, error) {
	var items []models.Item
	if err := config.DB.Find(&items).Error; err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	writer.Write(getCSVHeaders(role))
	for _, item := range items {
		writer.Write(formatItemCSVRow(item, role))
	}
	writer.Flush()

	return &dtos.CSVExport{
		FileName: "items.csv",
		Content:  buffer.Bytes(),
	}, nil
}


// Helper functions for CSV (internal to service)
func formatItemCSVRow(item models.Item, role string) []string {
	desc := common.GetStringValue(item.Description)
	img := common.GetStringValue(item.ImageURL)

	if role == "cashier" {
		return []string{
			fmt.Sprintf("%d", item.ID),
			item.Name,
			desc,
			fmt.Sprintf("%d", item.Stock),
			fmt.Sprintf("%.2f", item.Price),
			img,
		}
	}

	return []string{
		fmt.Sprintf("%d", item.ID),
		item.Name,
		desc,
		fmt.Sprintf("%d", item.Stock),
		fmt.Sprintf("%.2f", item.BuyPrice),
		fmt.Sprintf("%.2f", item.Price),
		img,
	}
}

func getCSVHeaders(role string) []string {
	if role == "cashier" {
		return []string{"id", "name", "description", "stock", "price", "image_url"}
	}
	return []string{"id", "name", "description", "stock", "buy_price", "price", "image_url"}
}
