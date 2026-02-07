package controllers

import (
	"net/http"
	"strconv"
	"time"

	"kd-api/config"
	"kd-api/models"
	"kd-api/utils/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

/* =========================
   OPEN CASH SESSION
   ========================= */
func OpenCashSession(c *gin.Context) {
	db := config.DB

	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := *userIDPtr

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

	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := *userIDPtr

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

	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := *userIDPtr

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

/* =========================
   GET CASH SESSION HISTORY
   ========================= */
func GetCashSessionHistory(c *gin.Context) {
	db := config.DB

	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := *userIDPtr

	// Parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize < 1 {
		pageSize = 10
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	offset := (page - 1) * pageSize

	var sessions []models.CashSession
	var total int64

	query := db.Model(&models.CashSession{}).Where("user_id = ?", userID)

	if startDate != "" {
		query = query.Where("opened_at >= ?", startDate+" 00:00:00")
	}

	if endDate != "" {
		query = query.Where("opened_at <= ?", endDate+" 23:59:59")
	}

	query.Count(&total)

	err := query.Order("opened_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&sessions).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       sessions,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": int(float64(total)/float64(pageSize) + 0.99),
	})
}
