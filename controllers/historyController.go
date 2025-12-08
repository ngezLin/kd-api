package controllers

import (
	"kd-api/config"
	"kd-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Get all transactions with pagination
func GetTransactions(c *gin.Context) {
    pageStr := c.DefaultQuery("page", "1")
    limitStr := c.DefaultQuery("limit", "10")
    filterDate := c.Query("date") // <--- AMBIL TANGGAL

    page, _ := strconv.Atoi(pageStr)
    limit, _ := strconv.Atoi(limitStr)

    if page < 1 {
        page = 1
    }
    if limit < 1 {
        limit = 10
    }
    offset := (page - 1) * limit

    var transactions []models.Transaction
    var total int64

    db := config.DB.Model(&models.Transaction{})

    // FILTER DATE JIKA ADA
    if filterDate != "" {
        // filter per hari
        start, _ := time.Parse("2006-01-02", filterDate)
        end := start.Add(24 * time.Hour)

        db = db.Where("created_at >= ? AND created_at < ?", start, end)
    }

    // hitung total
    if err := db.Count(&total).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // ambil data + preload
    if err := db.Preload("Items.Item").
        Order("created_at DESC").
        Limit(limit).
        Offset(offset).
        Find(&transactions).Error; err != nil {

        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":       transactions,
        "page":       page,
        "limit":      limit,
        "total":      total,
        "totalPages": int((total + int64(limit) - 1) / int64(limit)),
    })
}


// Get only completed + refunded transactions (history)
func GetTransactionHistory(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var transactions []models.Transaction
	var total int64

	if err := config.DB.Model(&models.Transaction{}).
		Where("status IN ?", []string{"completed", "refunded"}).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Preload("Items.Item").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       transactions,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": int((total + int64(limit) - 1) / int64(limit)),
	})
}

// Refund transaction (completed -> refunded, restore stock)
func RefundTransaction(c *gin.Context) {
	id := c.Param("id")

	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only completed transactions can be refunded"})
		return
	}

	for _, tItem := range transaction.Items {
		var item models.Item
		if err := config.DB.First(&item, tItem.ItemID).Error; err == nil {
			item.Stock += tItem.Quantity
			config.DB.Save(&item)
		}
	}

	transaction.Status = "refunded"
	transaction.Payment = nil
	transaction.Change = nil

	if err := config.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}