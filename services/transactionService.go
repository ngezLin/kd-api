package services

import (
	"errors"
	"fmt"
	"time"

	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"kd-api/utils/log"

	"gorm.io/gorm"
)

type TransactionService interface {
	CreateTransaction(input dtos.CreateTransactionInput, userID *uint, clientIP string) (*models.Transaction, []string, error)
	UpdateTransactionStatus(id string, input dtos.UpdateTransactionInput, userID *uint, clientIP string) (*models.Transaction, error)
	GetTransactions(filter dtos.TransactionFilter) (*dtos.TransactionListResponse, error)
	GetTransactionHistory(filter dtos.TransactionFilter) (*dtos.TransactionListResponse, error)
	GetTransactionByID(id string) (*models.Transaction, error)
	DeleteDraft(id string, userID *uint, clientIP string) error
	RefundTransaction(id string, userID *uint, clientIP string) (*models.Transaction, error)
}

type transactionService struct{}

func NewTransactionService() TransactionService {
	return &transactionService{}
}

func (s *transactionService) CreateTransaction(input dtos.CreateTransactionInput, userID *uint, clientIP string) (*models.Transaction, []string, error) {
	if len(input.Items) == 0 {
		return nil, nil, errors.New("no items provided")
	}

	if input.Status != "draft" && input.Status != "completed" {
		return nil, nil, errors.New("invalid transaction status")
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

			if i.Quantity == 0 {
				return fmt.Errorf("invalid quantity for item %d", i.ItemID)
			}

			price := item.Price
			if i.CustomPrice != nil {
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

		// Inventory Ledger: Log Sales
		if input.Status == "completed" {
			invService := NewInventoryService()
			for _, tItem := range transactionItems {
				// We need the ID, but we already have item.Stock updated.
				// Change is negative.
				change := -tItem.Quantity
				ref := fmt.Sprintf("TX-%d", transaction.ID)
				note := "Sold in transaction"
				
				if err := invService.LogStockChange(tx, tItem.ItemID, change, "sale", ref, userID, note); err != nil {
					return err
				}
			}
		}

		description := fmt.Sprintf("Transaction #%d created", transaction.ID)
		if err := log.CreateTransactionAuditLog(
			tx,
			"create",
			transaction.ID,
			nil,
			&transaction,
			userID,
			clientIP,
			description,
		); err != nil {
			return err
		}

		warnings = localWarnings
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	if err := config.DB.Preload("Items.Item").First(&transaction, transaction.ID).Error; err != nil {
		return nil, nil, err
	}

	return &transaction, warnings, nil
}

func (s *transactionService) UpdateTransactionStatus(id string, input dtos.UpdateTransactionInput, userID *uint, clientIP string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.Preload("Items").First(&transaction, id).Error; err != nil {
		return nil, errors.New("transaction not found")
	}

	oldCopy := transaction

	if input.Status != "" {
		if input.Status != "draft" && input.Status != "completed" {
			return nil, errors.New("invalid status")
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
		return nil, err
	}

	description := fmt.Sprintf("Transaction #%d updated", transaction.ID)
	if err := log.CreateTransactionAuditLog(
		config.DB,
		"update",
		transaction.ID,
		&oldCopy,
		&transaction,
		userID,
		clientIP,
		description,
	); err != nil {
		return nil, errors.New("failed to create audit log")
	}

	return &transaction, nil
}

func (s *transactionService) GetTransactions(filter dtos.TransactionFilter) (*dtos.TransactionListResponse, error) {
	var transactions []models.Transaction
	var total int64

	db := config.DB.Model(&models.Transaction{})

	if filter.Date != "" {
		start, _ := time.Parse("2006-01-02", filter.Date)
		end := start.Add(24 * time.Hour)
		db = db.Where("created_at >= ? AND created_at < ?", start, end)
	}

	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	if err := db.Preload("Items.Item").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	return &dtos.TransactionListResponse{
		Data:       transactions,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      total,
		TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
	}, nil
}

func (s *transactionService) GetTransactionHistory(filter dtos.TransactionFilter) (*dtos.TransactionListResponse, error) {
	var transactions []models.Transaction
	var total int64

	db := config.DB.Model(&models.Transaction{}).
		Where("status IN ?", []string{"completed", "refunded"})

	if filter.StartDate != "" {
		start, _ := time.Parse("2006-01-02", filter.StartDate)
		end := start.Add(24 * time.Hour)
		db = db.Where("created_at >= ? AND created_at < ?", start, end)
	}
	
	// Support for history by date specifically if Date is set but StartDate isn't
	if filter.Date != "" && filter.StartDate == "" {
		start, _ := time.Parse("2006-01-02", filter.Date)
		end := start.Add(24 * time.Hour)
		db = db.Where("created_at >= ? AND created_at < ?", start, end)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	if err := db.Preload("Items.Item").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, err
	}

	return &dtos.TransactionListResponse{
		Data:       transactions,
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      total,
		TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
	}, nil
}

func (s *transactionService) GetTransactionByID(id string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		return nil, errors.New("transaction not found")
	}
	return &transaction, nil
}

func (s *transactionService) DeleteDraft(id string, userID *uint, clientIP string) error {
	var transaction models.Transaction
	if err := config.DB.First(&transaction, id).Error; err != nil {
		return errors.New("transaction not found")
	}

	if transaction.Status != "draft" {
		return errors.New("only draft can be deleted")
	}

	txCopy := transaction

	if err := config.DB.Delete(&transaction).Error; err != nil {
		return errors.New("failed to delete")
	}

	description := fmt.Sprintf("Transaction #%d deleted", txCopy.ID)
	_ = log.CreateTransactionAuditLog(
		config.DB,
		"delete",
		txCopy.ID,
		&txCopy,
		nil,
		userID,
		clientIP,
		description,
	)

	return nil
}

func (s *transactionService) RefundTransaction(id string, userID *uint, clientIP string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.Preload("Items.Item").First(&transaction, id).Error; err != nil {
		return nil, errors.New("transaction not found")
	}

	if transaction.Status != "completed" {
		return nil, errors.New("only completed transactions can be refunded")
	}

	for _, tItem := range transaction.Items {
		var item models.Item
		if err := config.DB.First(&item, tItem.ItemID).Error; err == nil {
			item.Stock += tItem.Quantity
			config.DB.Save(&item)

			// Inventory Log (Refund)
			invService := NewInventoryService()
			change := tItem.Quantity // Refund is positive (stock returns)
			ref := fmt.Sprintf("TX-%d (REFUND)", transaction.ID)
			note := "Refunded transaction"

			invService.LogStockChange(config.DB, tItem.ItemID, change, "refund", ref, userID, note)
		}
	}

	transaction.Status = "refunded"
	transaction.Payment = nil
	transaction.Change = nil

	if err := config.DB.Save(&transaction).Error; err != nil {
		return nil, err
	}

	return &transaction, nil
}
