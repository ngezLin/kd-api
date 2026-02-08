package controllers

import (
	"net/http"

	"kd-api/dtos"
	"kd-api/services"
	"kd-api/utils/common"

	"github.com/gin-gonic/gin"
)


func CreateTransaction(c *gin.Context) {
	var input dtos.CreateTransactionInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewTransactionService()
	transaction, warnings, err := service.CreateTransaction(input, common.GetUserID(c), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{"transaction": transaction}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	c.JSON(http.StatusCreated, response)
}

func UpdateTransactionStatus(c *gin.Context) {
	id := c.Param("id")

	var input dtos.UpdateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := services.NewTransactionService()
	transaction, err := service.UpdateTransactionStatus(id, input, common.GetUserID(c), c.ClientIP())
	if err != nil {
		// Distinguish between not found and other errors if needed, but for now generic 500 or 400
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func GetDraftTransactions(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "draft"
	}

	service := services.NewTransactionService()
	response, err := service.GetTransactions(dtos.TransactionFilter{
		Status: status,
		Page:   1, // Default or query param
		Limit:  100, // Or query param, assuming fetch all drafts usually
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Maintains original response format which was a list of transactions
	c.JSON(http.StatusOK, response.Data)
}

func DeleteTransaction(c *gin.Context) {
	id := c.Param("id")
	service := services.NewTransactionService()
	
	err := service.DeleteDraft(id, common.GetUserID(c), c.ClientIP())
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "only draft can be deleted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Draft deleted"})
}

func GetTransactionByID(c *gin.Context) {
	id := c.Param("id")
	service := services.NewTransactionService()

	transaction, err := service.GetTransactionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if transaction.Status == "draft" {
		c.JSON(http.StatusOK, transaction)
		// Keeping the logic to delete draft after view? That seems odd in the original code but preserving behavior.
		// Wait, original code:
		// go func() { config.DB.Delete(&models.Transaction{}, id) }()
		// This deletes the draft immediately after viewing? 
		// I will replicate this behavior via service if needed, but maybe I should expose a method for it.
		// For now I'll call DeleteDraft in a goroutine ignoring errors as per original fire-and-forget.
		go func() {
			_ = service.DeleteDraft(id, common.GetUserID(c), c.ClientIP())
		}()
		return
	}

	c.JSON(http.StatusOK, transaction)
}
