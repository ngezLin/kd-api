package controllers

import (
	"net/http"
	"strconv"

	"kd-api/dtos"
	"kd-api/services"
	"kd-api/utils/common"

	"github.com/gin-gonic/gin"
)

/* =========================
   OPEN CASH SESSION
   ========================= */
func OpenCashSession(c *gin.Context) {
	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input dtos.OpenCashSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewCashSessionService()
	session, err := service.OpenCashSession(input, *userIDPtr)

	if err != nil {
		if err.Error() == "cash session masih terbuka" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

/* =========================
   GET CURRENT SESSION
   ========================= */
func GetCurrentCashSession(c *gin.Context) {
	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	service := services.NewCashSessionService()
	session, err := service.GetCurrentSession(*userIDPtr)

	if err != nil {
		if err.Error() == "tidak ada cash session aktif" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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
	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input dtos.CloseCashSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewCashSessionService()
	session, err := service.CloseCashSession(input, *userIDPtr)

	if err != nil {
		if err.Error() == "tidak ada cash session terbuka" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

/* =========================
   GET CASH SESSION HISTORY
   ========================= */
func GetCashSessionHistory(c *gin.Context) {
	userIDPtr := common.GetUserID(c)
	if userIDPtr == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	service := services.NewCashSessionService()
	response, err := service.GetSessionHistory(dtos.CashSessionHistoryFilter{
		Page:      page,
		PageSize:  pageSize,
		StartDate: startDate,
		EndDate:   endDate,
	}, *userIDPtr)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
