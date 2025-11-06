package controllers

import (
	"fmt"
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
		Discount        *float64 `json:"discount,omitempty"`
		Items           []struct {
			ItemID      uint     `json:"item_id"`
			Quantity    int      `json:"quantity"`
			CustomPrice *float64 `json:"customPrice,omitempty"`
		} `json:"items"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No items provided"})
		return
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var total float64
	var transactionItems []models.TransactionItem
	var warnings []string

	for _, i := range input.Items {
		var item models.Item
		if err := tx.First(&item, i.ItemID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Item %d not found", i.ItemID)})
			return
		}

		if i.Quantity <= 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid quantity for item %d", i.ItemID)})
			return
		}

		price := item.Price
		if i.CustomPrice != nil && *i.CustomPrice >= 0 {
			price = *i.CustomPrice
		}

		subtotal := float64(i.Quantity) * price
		total += subtotal

		transactionItems = append(transactionItems, models.TransactionItem{
			ItemID:   i.ItemID,
			Quantity: i.Quantity,
			Price:    price,
			Subtotal: subtotal,
		})
	}

	discount := 0.0
	if input.Discount != nil && *input.Discount > 0 {
		discount = *input.Discount
	}
	finalTotal := total - discount
	if finalTotal < 0 {
		finalTotal = 0
	}

	transaction := models.Transaction{
		Status:          input.Status,
		Total:           finalTotal,
		Discount:        discount,
		Items:           transactionItems,
		Note:            input.Note,
		TransactionType: "onsite",
	}

	if input.TransactionType != nil && *input.TransactionType != "" {
		transaction.TransactionType = *input.TransactionType
	}

	if input.Status == "completed" {
		if input.PaymentAmount == nil || *input.PaymentAmount < finalTotal {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not enough"})
			return
		}

		change := *input.PaymentAmount - finalTotal
		transaction.Payment = input.PaymentAmount
		transaction.Change = &change

		if input.PaymentType != nil && *input.PaymentType != "" {
			transaction.PaymentType = input.PaymentType
		} else {
			defaultType := "cash"
			transaction.PaymentType = &defaultType
		}

		// Kurangi stok, tapi kalau stok kurang → tetap jalan + beri warning
		for _, tItem := range transactionItems {
			var item models.Item
			if err := tx.First(&item, tItem.ItemID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Item %d not found", tItem.ItemID)})
				return
			}

			if item.Stock < tItem.Quantity {
				warnings = append(warnings,
					fmt.Sprintf("Warning: Item '%s' stock insufficient (current: %d, required: %d)", item.Name, item.Stock, tItem.Quantity),
				)
			}

			item.Stock -= tItem.Quantity
			if err := tx.Save(&item).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Preload("Items.Item").First(&transaction, transaction.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"transaction": transaction,
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	c.JSON(http.StatusCreated, response)
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
	if err := config.DB.Preload("Items").First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	var input struct {
		Status          string   `json:"status"`
		Note            *string  `json:"note,omitempty"`
		TransactionType *string  `json:"transaction_type,omitempty"`
		Discount        *float64 `json:"discount,omitempty"`
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

	// Apply discount if provided
	if input.Discount != nil {
		if *input.Discount < 0 {
			transaction.Discount = 0
		} else {
			transaction.Discount = *input.Discount
		}

		// Recalculate total after discount
		var total float64
		for _, tItem := range transaction.Items {
			total += tItem.Subtotal
		}
		finalTotal := total - transaction.Discount
		if finalTotal < 0 {
			finalTotal = 0
		}
		transaction.Total = finalTotal
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
