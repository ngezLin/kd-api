package controllers

import (
	"kd-api/dtos"
	"kd-api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Get all transactions
func GetTransactions(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    filterDate := c.Query("date")

    service := services.NewTransactionService()
    response, err := service.GetTransactions(dtos.TransactionFilter{
        Page:  page,
        Limit: limit,
        Date:  filterDate,
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, response)
}


// Get only completed + refunded transactions (history)
func GetTransactionHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

    service := services.NewTransactionService()
    response, err := service.GetTransactionHistory(dtos.TransactionFilter{
        Page:  page,
        Limit: limit,
    })

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Refund transaction (completed -> refunded, restore stock)
func RefundTransaction(c *gin.Context) {
	id := c.Param("id")
    service := services.NewTransactionService()
    
    transaction, err := service.RefundTransaction(id)
    if err != nil {
        if err.Error() == "transaction not found" {
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
            return
        }
        if err.Error() == "only completed transactions can be refunded" {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

	c.JSON(http.StatusOK, transaction)
}

func GetTransactionHistoryByDate(c *gin.Context) {
    pageStr := c.DefaultQuery("page", "1")
    limitStr := c.DefaultQuery("limit", "10")
    filterDate := c.Query("date") // tanggal wajib

    if filterDate == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "date query param is required (format: YYYY-MM-DD)"})
        return
    }

    page, _ := strconv.Atoi(pageStr)
    limit, _ := strconv.Atoi(limitStr)

    service := services.NewTransactionService()
    response, err := service.GetTransactionHistory(dtos.TransactionFilter{
        Page:  page,
        Limit: limit,
        Date:  filterDate, // Reusing Date field if StartDate logic handles it or adding simpler handling in service
    })

     if err != nil {
        if err.Error() == "Invalid date format, use YYYY-MM-DD" {
             c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
             return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, response)
}
