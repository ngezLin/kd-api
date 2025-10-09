package controllers

import (
	"net/http"

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

// Get all items
func GetItems(c *gin.Context) {
	var items []models.Item
	if err := config.DB.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	role := getUserRole(c)
	c.JSON(http.StatusOK, filterItemsForRole(items, role))
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