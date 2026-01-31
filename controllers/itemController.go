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
	"kd-api/utils/common"
	"kd-api/utils/log"
	"kd-api/utils/pagination"
	"kd-api/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func GetItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	p := pagination.New(page, pageSize)

	var items []models.Item
	var total int64

	if err := config.DB.Model(&models.Item{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.
		Offset(p.Offset).
		Limit(p.PageSize).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	meta := pagination.BuildMeta(p.Page, p.PageSize, total)

	c.JSON(http.StatusOK, gin.H{
		"data": response.FilterItemsForRole(items, common.GetUserRole(c)),
		"meta": meta,
	})
}


func GetItemsByName(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name parameter is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	p := pagination.New(page, pageSize)

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

	if err := query.
		Offset(p.Offset).
		Limit(p.PageSize).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	meta := pagination.BuildMeta(p.Page, p.PageSize, total)

	c.JSON(http.StatusOK, gin.H{
		"data": response.FilterItemsForRole(items, common.GetUserRole(c)),
		"meta": meta,
	})
}


func GetItemByID(c *gin.Context) {
	var item models.Item
	if err := config.DB.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, response.FilterItemForRole(item, common.GetUserRole(c)))
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
		return log.CreateItemAuditLog(
			tx,
			"create",
			input.ID,
			nil,
			&input,
			common.GetUserID(c),
			c.ClientIP(),
			description,
		)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.FilterItemForRole(input, common.GetUserRole(c)))
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
		return log.CreateItemAuditLog(
			tx,
			"update",
			oldItem.ID,
			&oldCopy,
			&oldItem,
			common.GetUserID(c),
			c.ClientIP(),
			description,
		)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.FilterItemForRole(oldItem, common.GetUserRole(c)))
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
		return log.CreateItemAuditLog(
			tx,
			"delete",
			itemCopy.ID,
			&itemCopy,
			nil,
			common.GetUserID(c),
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

	userID := common.GetUserID(c)
	ipAddress := c.ClientIP()

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&inputs).Error; err != nil {
			return err
		}

		for _, item := range inputs {
			description := fmt.Sprintf("Item '%s' created via bulk import", item.Name)
			if err := log.CreateItemAuditLog(
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

	c.JSON(http.StatusCreated, response.FilterItemsForRole(inputs, common.GetUserRole(c)))
}

func ExportItems(c *gin.Context) {
	var items []models.Item
	if err := config.DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := common.GetUserRole(c)
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	writer.Write(getCSVHeaders(role))
	for _, item := range items {
		writer.Write(formatItemCSVRow(item, role))
	}
	writer.Flush()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="items.csv"`)
	c.Data(http.StatusOK, "text/csv", buffer.Bytes())
}