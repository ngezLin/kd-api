package controllers

import (
	"strconv"

	"net/http"

	"kd-api/dtos"
	"kd-api/models"
	"kd-api/services"
	"kd-api/utils/common"

	"github.com/gin-gonic/gin"
)

func GetItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	service := services.NewItemService()
	response, err := service.GetItems(dtos.ItemFilter{
		Page:     page,
		PageSize: pageSize,
	}, common.GetUserRole(c))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response.Data,
		"meta": response.Meta,
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

	service := services.NewItemService()
	response, err := service.GetItems(dtos.ItemFilter{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
	}, common.GetUserRole(c))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response.Data,
		"meta": response.Meta,
	})
}


func GetItemByID(c *gin.Context) {
	service := services.NewItemService()
	item, err := service.GetItemByID(c.Param("id"), common.GetUserRole(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func CreateItem(c *gin.Context) {
	var input dtos.CreateItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewItemService()
	item, err := service.CreateItem(input, common.GetUserID(c), c.ClientIP(), common.GetUserRole(c))
	
	if err != nil {
		if err.Error() == "Item dengan nama ini sudah ada" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func UpdateItem(c *gin.Context) {
	var input dtos.UpdateItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewItemService()
	item, err := service.UpdateItem(c.Param("id"), input, common.GetUserID(c), c.ClientIP(), common.GetUserRole(c))

	if err != nil {
		if err.Error() == "Item not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "Item dengan nama ini sudah ada" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

func DeleteItem(c *gin.Context) {
	service := services.NewItemService()
	err := service.DeleteItem(c.Param("id"), common.GetUserID(c), c.ClientIP())

	if err != nil {
		if err.Error() == "Item not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
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

	service := services.NewItemService()
	items, err := service.BulkCreateItems(inputs, common.GetUserID(c), c.ClientIP(), common.GetUserRole(c))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, items)
}

func ExportItems(c *gin.Context) {
	service := services.NewItemService()
	export, err := service.ExportItems(common.GetUserRole(c))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=\"" + export.FileName + "\"")
	c.Data(http.StatusOK, "text/csv", export.Content)
}
