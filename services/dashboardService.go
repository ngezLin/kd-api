package services

import (
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"time"
)

type DashboardService interface {
	GetDashboardStats() (*dtos.DashboardStats, error)
}

type dashboardService struct{}

func NewDashboardService() DashboardService {
	return &dashboardService{}
}

func (s *dashboardService) GetDashboardStats() (*dtos.DashboardStats, error) {
	var todayProfit float64
	var todayTransactions int64
	var lowStock int64
	var topItems []dtos.TopItem

	today := time.Now().Format("2006-01-02")
	var todayTransactionsData []models.Transaction

	// Calculate today's profit
	if err := config.DB.Preload("Items.Item").
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Find(&todayTransactionsData).Error; err != nil {
		return nil, err
	}

	for _, t := range todayTransactionsData {
		for _, ti := range t.Items {
			todayProfit += float64(ti.Quantity) * (ti.Price - ti.Item.BuyPrice)
		}
	}

	// Count today's transactions
	if err := config.DB.Model(&models.Transaction{}).
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Count(&todayTransactions).Error; err != nil {
		return nil, err
	}

	// Count low stock items (<5)
	if err := config.DB.Model(&models.Item{}).Where("stock < ?", 5).Count(&lowStock).Error; err != nil {
		return nil, err
	}

	// Get top selling items (top 5)
	if err := config.DB.Model(&models.TransactionItem{}).
		Select("item_id, SUM(quantity) as quantity").
		Joins("JOIN transactions ON transactions.id = transaction_items.transaction_id").
		Where("transactions.status = ?", "completed").
		Group("item_id").
		Order("quantity desc").
		Limit(5).
		Scan(&topItems).Error; err != nil {
		return nil, err
	}

	// Populate item names for top items
	for i, ti := range topItems {
		var item models.Item
		if err := config.DB.First(&item, ti.ItemID).Error; err == nil {
			topItems[i].Name = item.Name
		}
	}

	return &dtos.DashboardStats{
		TodayProfit:       todayProfit,
		TodayTransactions: todayTransactions,
		LowStock:          lowStock,
		TopSellingItems:   topItems,
	}, nil
}
