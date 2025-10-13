package controllers

import (
	"net/http"

	"kd-api/config"
	"kd-api/models"

	"github.com/gin-gonic/gin"
)

// Create new transaction (draft / completed)
func CreateTransaction(c *gin.Context) {
	var input struct {
		Status          string   `json:"status"` 
		PaymentAmount   *float64 `json:"paymentAmount,omitempty"`
		PaymentType     *string  `json:"paymentType,omitempty"`
		Note            *string  `json:"note,omitempty"`
		TransactionType *string  `json:"transaction_type,omitempty"` 
		Items           []struct {
			ItemID   uint `json:"item_id"`
			Quantity int  `json:"quantity"`
		} `json:"items"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var total float64
	var transactionItems []models.TransactionItem

	for _, i := range input.Items {
		var item models.Item
		if err := config.DB.First(&item, i.ItemID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Item not found"})
			return
		}

		subtotal := float64(i.Quantity) * item.Price
		total += subtotal

		transactionItems = append(transactionItems, models.TransactionItem{
			ItemID:   i.ItemID,
			Quantity: i.Quantity,
			Price:    item.Price,
			Subtotal: subtotal,
		})
	}

	transaction := models.Transaction{
		Status:          input.Status,
		Total:           total,
		Items:           transactionItems,
		Note:            input.Note,
		TransactionType: "onsite", // default
	}

	// Jika ada TransactionType dari input, pakai itu
	if input.TransactionType != nil {
		transaction.TransactionType = *input.TransactionType
	}

	// Jika completed, hitung change dan validasi payment
	if input.Status == "completed" {
		if input.PaymentAmount == nil || *input.PaymentAmount < total {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not enough"})
			return
		}
		change := *input.PaymentAmount - total
		transaction.Payment = input.PaymentAmount
		transaction.Change = &change
		transaction.PaymentType = input.PaymentType

		// Kurangi stock item
		for _, tItem := range transaction.Items {
			var item models.Item
			if err := config.DB.First(&item, tItem.ItemID).Error; err == nil {
				item.Stock -= tItem.Quantity
				config.DB.Save(&item)
			}
		}
	}

	if err := config.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Preload("Items.Item").First(&transaction, transaction.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// Get all transactions
func GetTransactions(c *gin.Context) {
	var transactions []models.Transaction
	if err := config.DB.Preload("Items.Item").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// Get transaction by ID
func GetTransactionByID(c *gin.Context) {
	id := c.Param("id")
	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	// Jika draft, hapus setelah dikirim ke frontend
	if transaction.Status == "draft" {
		c.JSON(http.StatusOK, transaction)
		go func(id string) {
			config.DB.Delete(&models.Transaction{}, id)
		}(id)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// Update status, note, dan transaction type
func UpdateTransactionStatus(c *gin.Context) {
	id := c.Param("id")
	var transaction models.Transaction
	if err := config.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	var input struct {
		Status          string  `json:"status"`
		Note            *string `json:"note,omitempty"`
		TransactionType *string `json:"transaction_type,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction.Status = input.Status
	if input.Note != nil {
		transaction.Note = input.Note
	}
	if input.TransactionType != nil {
		transaction.TransactionType = *input.TransactionType
	}

	if err := config.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// Get only completed + refunded transactions (history)
func GetTransactionHistory(c *gin.Context) {
	var transactions []models.Transaction
	if err := config.DB.Preload("Items.Item").
		Where("status IN ?", []string{"completed", "refunded"}).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// Checkout transaction (draft -> completed, reduce stock)
func CheckoutTransaction(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		TransactionType *string  `json:"transaction_type,omitempty"`
		PaymentAmount   *float64 `json:"paymentAmount,omitempty"`
		PaymentType     *string  `json:"paymentType,omitempty"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only draft transactions can be checked out"})
		return
	}

	for _, tItem := range transaction.Items {
		var item models.Item
		if err := config.DB.First(&item, tItem.ItemID).Error; err == nil {
			item.Stock -= tItem.Quantity
			config.DB.Save(&item)
		}
	}

	transaction.Status = "completed"

	if input.PaymentAmount != nil {
		transaction.Payment = input.PaymentAmount
		change := *input.PaymentAmount - transaction.Total
		transaction.Change = &change
	}

	if input.PaymentType != nil {
		transaction.PaymentType = input.PaymentType
	}

	if input.TransactionType != nil {
		transaction.TransactionType = *input.TransactionType
	}

	if err := config.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
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

// GET /transactions?status=draft
func GetDraftTransactions(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "draft"
	}
	var transactions []models.Transaction
	if err := config.DB.Preload("Items.Item").Where("status = ?", status).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// DELETE /transactions/:id
func DeleteTransaction(c *gin.Context) {
	id := c.Param("id")

	var transaction models.Transaction
	if err := config.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only draft can be deleted"})
		return
	}

	if err := config.DB.Delete(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Draft deleted"})
}
