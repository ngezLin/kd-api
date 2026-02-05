package controllers

import (
	"net/http"
	"time"

	"kd-api/config"
	"kd-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

/* =========================
   HELPER: ambil user_id aman
   ========================= */
func getUserID(c *gin.Context) (uint, bool) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return 0, false
	}

	userID, ok := userIDAny.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id",
		})
		return 0, false
	}

	return userID, true
}

/* =========================
   OPEN CASH SESSION
   ========================= */
func OpenCashSession(c *gin.Context) {
	db := config.DB

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var input struct {
		OpeningCash float64 `json:"opening_cash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.CashSession
	err := db.Where("user_id = ? AND status = 'open'", userID).
		First(&existing).Error

	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cash session masih terbuka",
		})
		return
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

/* =========================
   GET CURRENT SESSION
   ========================= */
func GetCurrentCashSession(c *gin.Context) {
	db := config.DB

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var session models.CashSession
	if err := db.Where("user_id = ? AND status = 'open'", userID).
		First(&session).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "tidak ada cash session aktif",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

/* =========================
   CLOSE CASH SESSION
   ========================= */
func CloseCashSession(c *gin.Context) {
	db := config.DB

	userID, ok := getUserID(c)
	if !ok {
		return
	}

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

	/* =========================
	   HITUNG TRANSAKSI CASH
	   ========================= */
	var result struct {
		TotalCashIn float64
		TotalChange float64
	}

	db.Model(&models.Transaction{}).
		Select(
			"COALESCE(SUM(payment), 0) AS total_cash_in, " +
				"COALESCE(SUM(`change`), 0) AS total_change",
		).
		Where(
			"payment_type = ? AND status = ? AND created_at BETWEEN ? AND ?",
			"cash", "completed", session.OpenedAt, time.Now(),
		).
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
