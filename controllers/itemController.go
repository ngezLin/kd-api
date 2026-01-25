package controllers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"kd-api/config"
	"kd-api/models"
	"kd-api/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

func getPaginationParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return
}

func buildPaginatedResponse(items []models.Item, page, pageSize int, total int64, role string) PaginatedResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       utils.FilterItemsForRole(items, role),
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

func GetItems(c *gin.Context) {
	page, pageSize := getPaginationParams(c)

	var items []models.Item
	var total int64

	if err := config.DB.Model(&models.Item{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize
	if err := config.DB.Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildPaginatedResponse(items, page, pageSize, total, utils.GetUserRole(c)))
}

func GetItemsByName(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name parameter is required"})
		return
	}

	page, pageSize := getPaginationParams(c)

	var items []models.Item
	var total int64

	query := config.DB.Model(&models.Item{})
	for _, term := range strings.Fields(strings.ToLower(strings.TrimSpace(name))) {
		query = query.Where("LOWER(name) LIKE ?", "%"+term+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, buildPaginatedResponse(items, page, pageSize, total, utils.GetUserRole(c)))
}

func GetItemByID(c *gin.Context) {
	var item models.Item
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, utils.FilterItemForRole(item, utils.GetUserRole(c)))
}

func CreateItem(c *gin.Context) {
	var input models.Item
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Item
	if err := config.DB.Where("name = ?", input.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item dengan nama ini sudah ada"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&input).Error; err != nil {
			return err
		}

		description := fmt.Sprintf("Item '%s' created", input.Name)
		return utils.CreateItemAuditLog(
			tx,
			"create",
			input.ID,
			nil,
			&input,
			utils.GetUserID(c),
			c.ClientIP(),
			description,
		)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, utils.FilterItemForRole(input, utils.GetUserRole(c)))
}

func UpdateItem(c *gin.Context) {
	var oldItem models.Item
	if err := config.DB.First(&oldItem, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	var input models.Item
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Item
	if err := config.DB.Where("name = ? AND id != ?", input.Name, oldItem.ID).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item dengan nama ini sudah ada"})
		return
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
		return utils.CreateItemAuditLog(
			tx,
			"update",
			oldItem.ID,
			&oldCopy,
			&oldItem,
			utils.GetUserID(c),
			c.ClientIP(),
			description,
		)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, utils.FilterItemForRole(oldItem, utils.GetUserRole(c)))
}

func DeleteItem(c *gin.Context) {
	var item models.Item
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	itemCopy := item

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&item).Error; err != nil {
			return err
		}

		description := fmt.Sprintf("Item '%s' deleted", itemCopy.Name)
		return utils.CreateItemAuditLog(
			tx,
			"delete",
			itemCopy.ID,
			&itemCopy,
			nil,
			utils.GetUserID(c),
			c.ClientIP(),
			description,
		)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item deleted successfully"})
}

func BulkCreateItems(c *gin.Context) {
	var inputs []models.Item
	if err := c.ShouldBindJSON(&inputs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for i := range inputs {
		if inputs[i].Description != nil && *inputs[i].Description == "" {
			inputs[i].Description = nil
		}
		if inputs[i].ImageURL != nil && *inputs[i].ImageURL == "" {
			inputs[i].ImageURL = nil
		}
	}

	userID := utils.GetUserID(c)
	ipAddress := c.ClientIP()

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&inputs).Error; err != nil {
			return err
		}

		for _, item := range inputs {
			description := fmt.Sprintf("Item '%s' created via bulk import", item.Name)
			if err := utils.CreateItemAuditLog(
				tx,
				"create",
				item.ID,
				nil,
				&item,
				userID,
				ipAddress,
				description,
			); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, utils.FilterItemsForRole(inputs, utils.GetUserRole(c)))
}

func ExportItems(c *gin.Context) {
	var items []models.Item
	if err := config.DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := utils.GetUserRole(c)
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	writer.Write(utils.GetCSVHeaders(role))
	for _, item := range items {
		writer.Write(utils.FormatItemCSVRow(item, role))
	}
	writer.Flush()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="items.csv"`)
	c.Data(http.StatusOK, "text/csv", buffer.Bytes())
}