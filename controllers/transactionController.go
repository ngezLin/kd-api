package controllers

import (
	"fmt"
	"net/http"

	"kd-api/config"
	"kd-api/models"
	"kd-api/utils"

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
	var itemNames []string // Untuk notifikasi WA

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

		// Simpan nama item untuk notifikasi
		itemNames = append(itemNames, fmt.Sprintf("%s x%d", item.Name, i.Quantity))

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

	// KIRIM NOTIFIKASI WHATSAPP (jangan blocking)
	go func() {
		message := utils.FormatTransactionMessage(
			transaction.ID,
			transaction.Status,
			transaction.Total,
			itemNames,
		)
		
		err := utils.SendWhatsAppNotification("081357022138", message)
		if err != nil {
			// Log error tapi jangan ganggu response API
			fmt.Printf("Failed to send WhatsApp notification: %v\n", err)
		}
	}()

	response := gin.H{
		"transaction": transaction,
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	c.JSON(http.StatusCreated, response)
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
