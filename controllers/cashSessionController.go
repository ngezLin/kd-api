package controllers

import (
	"net/http"
	"time"

	"kd-api/config"
	"kd-api/models"

	"github.com/gin-gonic/gin"
)

func OpenCashSession(c *gin.Context) {
	db := config.DB

	userID := c.MustGet("user_id").(uint)

	var input struct {
		OpeningCash float64 `json:"opening_cash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.CashSession
	if err := db.Where("user_id = ? AND status = 'open'", userID).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cash session masih terbuka",
		})
		return
	}

	session := models.CashSession{
		UserID:       userID,
		OpeningCash: input.OpeningCash,
		Status:       "open",
		OpenedAt:     time.Now(),
	}

	if err := db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func GetCurrentCashSession(c *gin.Context) {
	db := config.DB
	userID := c.MustGet("user_id").(uint)

	var session models.CashSession
	if err := db.Where("user_id = ? AND status = 'open'", userID).
		First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "tidak ada cash session aktif",
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

func CloseCashSession(c *gin.Context) {
	db := config.DB
	userID := c.MustGet("user_id").(uint)

	var input struct {
		ClosingCash float64 `json:"closing_cash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var session models.CashSession
	if err := db.Where("user_id = ? AND status = 'open'", userID).
		First(&session).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tidak ada cash session terbuka",
		})
		return
	}

	// Hitung transaksi CASH dalam range waktu
	var result struct {
		TotalCashIn float64
		TotalChange float64
	}

	db.Model(&models.Transaction{}).
		Select(`
			COALESCE(SUM(payment), 0) as total_cash_in,
			COALESCE(SUM(change), 0) as total_change
		`).
		Where(`
			payment_type = 'cash'
			AND status = 'completed'
			AND created_at BETWEEN ? AND ?
		`, session.OpenedAt, time.Now()).
		Scan(&result)

	expected := session.OpeningCash +
		result.TotalCashIn -
		result.TotalChange

	diff := input.ClosingCash - expected

	session.TotalCashIn = result.TotalCashIn
	session.TotalChange = result.TotalChange
	session.ExpectedCash = expected
	session.ClosingCash = &input.ClosingCash
	session.Difference = &diff
	session.Status = "closed"
	now := time.Now()
	session.ClosedAt = &now

	if err := db.Save(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}
