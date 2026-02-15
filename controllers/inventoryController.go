package controllers

import (
	"kd-api/dtos"
	"kd-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetInventoryHistory(c *gin.Context) {
	var filter dtos.InventoryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewInventoryService()
	response, err := service.GetInventoryHistory(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
