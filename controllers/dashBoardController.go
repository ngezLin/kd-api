package controllers

import (
	"kd-api/config"
	"kd-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardTransactionItem struct {
	ItemID   uint    `json:"item_id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Subtotal float64 `json:"subtotal"`
}

type DashboardTransaction struct {
	ID     uint                       `json:"id"`
	Status string                     `json:"status"`
	Items  []DashboardTransactionItem `json:"items"`
}

type TopItem struct {
	ItemID   uint   `json:"item_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

func GetDashboard(c *gin.Context) {
	var todayProfit float64
	var todayTransactions int64

	// Hitung profit hari ini
	today := time.Now().Format("2006-01-02")
	var todayTransactionsData []models.Transaction

	config.DB.Preload("Items.Item").
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Find(&todayTransactionsData)

	for _, t := range todayTransactionsData {
		for _, ti := range t.Items {
			todayProfit += float64(ti.Quantity) * (ti.Price - ti.Item.BuyPrice)
		}
	}

	// Hitung jumlah transaksi hari ini
	config.DB.Model(&models.Transaction{}).
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Count(&todayTransactions)


	// Low stock count (<5)
	var lowStock int64
	config.DB.Model(&models.Item{}).Where("stock < ?", 5).Count(&lowStock)

	// Top selling items (top 5)
	var topItems []TopItem
	config.DB.Model(&models.TransactionItem{}).
		Select("item_id, SUM(quantity) as quantity").
		Joins("JOIN transactions ON transactions.id = transaction_items.transaction_id").
		Where("transactions.status = ?", "completed").
		Group("item_id").
		Order("quantity desc").
		Limit(5).
		Scan(&topItems)

	for i, ti := range topItems {
		var item models.Item
		if err := config.DB.First(&item, ti.ItemID).Error; err == nil {
			topItems[i].Name = item.Name
		}
	}

	// Kirim response JSON
	c.JSON(http.StatusOK, gin.H{
		"today_profit":        todayProfit,
		"today_transactions":  todayTransactions,
		"low_stock":           lowStock,
		"top_selling_items":   topItems,
	})
}
