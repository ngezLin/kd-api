package controllers

import (
	"net/http"
	"strconv"

	"kd-api/config"
	"kd-api/models"

	"github.com/gin-gonic/gin"
)

func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var logs []models.AuditLog
	var total int64

	query := config.DB.Model(&models.AuditLog{})

	// optional filters
	if entity := c.Query("entity_type"); entity != "" {
		query = query.Where("entity_type = ?", entity)
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        logs,
		"page":        page,
		"page_size":   pageSize,
		"total_items": total,
	})
}