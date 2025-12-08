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

	"github.com/gin-gonic/gin"
)

// ItemResponse for cashier (without BuyPrice)
type ItemResponseCashier struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Stock       int     `json:"stock"`
	Price       float64 `json:"price"`
	ImageURL    *string `json:"image_url,omitempty"`
}

// Pagination response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

// Helper function to get user role from context
func getUserRole(c *gin.Context) string {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	return role.(string)
}

// Helper function to filter items based on role
func filterItemsForRole(items []models.Item, role string) interface{} {
	if role == "cashier" {
		cashierItems := make([]ItemResponseCashier, len(items))
		for i, item := range items {
			cashierItems[i] = ItemResponseCashier{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Stock:       item.Stock,
				Price:       item.Price,
				ImageURL:    item.ImageURL,
			}
		}
		return cashierItems
	}
	return items
}

// Helper function to filter single item based on role
func filterItemForRole(item models.Item, role string) interface{} {
	if role == "cashier" {
		return ItemResponseCashier{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Stock:       item.Stock,
			Price:       item.Price,
			ImageURL:    item.ImageURL,
		}
	}
	return item
}

// Get all items with pagination
func GetItems(c *gin.Context) {
	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var items []models.Item
	var totalItems int64

	// Count total items
	if err := config.DB.Model(&models.Item{}).Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated items
	if err := config.DB.Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize != 0 {
		totalPages++
	}

	role := getUserRole(c)
	filteredItems := filterItemsForRole(items, role)

	response := PaginatedResponse{
		Data:       filteredItems,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// Get items by name with pagination (supports fuzzy search)
func GetItemsByName(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name parameter is required"})
		return
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var items []models.Item
	var totalItems int64

	// Build fuzzy search query
	// Split search terms by whitespace and create LIKE conditions for each word
	searchTerms := strings.Fields(strings.TrimSpace(strings.ToLower(name)))
	
	query := config.DB.Model(&models.Item{})
	
	// Each search term must appear somewhere in the name (case-insensitive)
	for _, term := range searchTerms {
		query = query.Where("LOWER(name) LIKE ?", "%"+term+"%")
	}

	// Count total items matching the search
	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated items
	if err := query.Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total pages
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize != 0 {
		totalPages++
	}

	role := getUserRole(c)
	filteredItems := filterItemsForRole(items, role)

	response := PaginatedResponse{
		Data:       filteredItems,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// Get item by ID
func GetItemByID(c *gin.Context) {
	id := c.Param("id")
	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	
	role := getUserRole(c)
	c.JSON(http.StatusOK, filterItemForRole(item, role))
}

// Create new item
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

	item := models.Item{
		Name:        input.Name,
		Description: input.Description,
		Stock:       input.Stock,
		BuyPrice:    input.BuyPrice,
		Price:       input.Price,
		ImageURL:    input.ImageURL,
	}

	if err := config.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := getUserRole(c)
	c.JSON(http.StatusCreated, filterItemForRole(item, role))
}

// Update item by ID
func UpdateItem(c *gin.Context) {
	id := c.Param("id")
	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	var input models.Item
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Item
	if err := config.DB.Where("name = ? AND id != ?", input.Name, id).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item dengan nama ini sudah ada"})
		return
	}

	item.Name = input.Name
	item.Description = input.Description
	item.Stock = input.Stock
	item.BuyPrice = input.BuyPrice
	item.Price = input.Price
	item.ImageURL = input.ImageURL

	if err := config.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := getUserRole(c)
	c.JSON(http.StatusOK, filterItemForRole(item, role))
}

// Delete item by ID
func DeleteItem(c *gin.Context) {
	id := c.Param("id")
	var item models.Item
	if err := config.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if err := config.DB.Delete(&item).Error; err != nil {
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
		// Konversi string kosong menjadi nil untuk pointer fields
		if inputs[i].Description != nil && *inputs[i].Description == "" {
			inputs[i].Description = nil
		}
		if inputs[i].ImageURL != nil && *inputs[i].ImageURL == "" {
			inputs[i].ImageURL = nil
		}
	}

	if err := config.DB.Create(&inputs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := getUserRole(c)
	c.JSON(http.StatusCreated, filterItemsForRole(inputs, role))
}

func ExportItems(c *gin.Context) {
	var items []models.Item

	if err := config.DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role := getUserRole(c)

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Header CSV

	writer.Write([]string{
		"id", "name", "description", "stock", "buy_price", "price", "image_url",
	})

	// Rows
	for _, item := range items {
		desc := ""
		img := ""

		if item.Description != nil {
			desc = *item.Description
		}
		if item.ImageURL != nil {
			img = *item.ImageURL
		}

		if role == "cashier" {
			writer.Write([]string{
				fmt.Sprintf("%d", item.ID),
				item.Name,
				desc,
				fmt.Sprintf("%d", item.Stock),
				fmt.Sprintf("%.2f", item.Price),
				img,
			})
		} else {
			writer.Write([]string{
				fmt.Sprintf("%d", item.ID),
				item.Name,
				desc,
				fmt.Sprintf("%d", item.Stock),
				fmt.Sprintf("%.2f", item.BuyPrice),
				fmt.Sprintf("%.2f", item.Price),
				img,
			})
		}
	}

	writer.Flush()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="items.csv"`)
	c.Data(http.StatusOK, "text/csv", buffer.Bytes())
}