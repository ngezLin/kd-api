package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"kd-api/config"
	"kd-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	if input.Status != "draft" && input.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction status"})
		return
	}

	var transaction models.Transaction
	var warnings []string

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var total float64
		var transactionItems []models.TransactionItem
		var localWarnings []string

		for _, i := range input.Items {
			var item models.Item
			if err := tx.First(&item, i.ItemID).Error; err != nil {
				return fmt.Errorf("item %d not found", i.ItemID)
			}

			if i.Quantity <= 0 {
				return fmt.Errorf("invalid quantity for item %d", i.ItemID)
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

		transaction = models.Transaction{
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
				return errors.New("payment not enough")
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

			for _, tItem := range transactionItems {
				var item models.Item
				if err := tx.First(&item, tItem.ItemID).Error; err != nil {
					return err
				}

				if item.Stock < tItem.Quantity {
					localWarnings = append(localWarnings,
						fmt.Sprintf(
							"Warning: Item '%s' stock insufficient (current: %d, required: %d)",
							item.Name, item.Stock, tItem.Quantity,
						),
					)
					item.Stock = 0
				} else {
					item.Stock -= tItem.Quantity
				}

				if err := tx.Save(&item).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		warnings = localWarnings
		return nil
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Preload("Items.Item").First(&transaction, transaction.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	if input.Status != "" {
		if input.Status != "draft" && input.Status != "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		transaction.Status = input.Status
	}

	if input.Note != nil {
		transaction.Note = input.Note
	}

	if input.TransactionType != nil {
		transaction.TransactionType = *input.TransactionType
	}

	if input.Discount != nil {
		if *input.Discount < 0 {
			transaction.Discount = 0
		} else {
			transaction.Discount = *input.Discount
		}

		var total float64
		for _, item := range transaction.Items {
			total += item.Subtotal
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

func GetDraftTransactions(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "draft"
	}

	var transactions []models.Transaction
	if err := config.DB.Preload("Items.Item").
		Where("status = ?", status).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

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

func GetTransactionByID(c *gin.Context) {
	id := c.Param("id")

	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	if transaction.Status == "draft" {
		c.JSON(http.StatusOK, transaction)
		go func() {
			config.DB.Delete(&models.Transaction{}, id)
		}()
		return
	}

	c.JSON(http.StatusOK, transaction)
}
