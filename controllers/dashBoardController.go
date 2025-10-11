package controllers

import (
	"fmt"
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
	ID     uint                      `json:"id"`
	Status string                    `json:"status"`
	Items  []DashboardTransactionItem `json:"items"`
}

type TopItem struct {
	ItemID   uint   `json:"item_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

func GetDashboard(c *gin.Context) {
	var totalItems int64
	var totalTransactions int64
	var totalDraft int64
	var totalCompleted int64
	var totalRefunded int64
	var totalOmzet float64
	var totalProfit float64
	var todayProfit float64

	// Count items
	config.DB.Model(&models.Item{}).Count(&totalItems)

	// Count transactions
	config.DB.Model(&models.Transaction{}).Count(&totalTransactions)
	config.DB.Model(&models.Transaction{}).Where("status = ?", "draft").Count(&totalDraft)
	config.DB.Model(&models.Transaction{}).Where("status = ?", "completed").Count(&totalCompleted)
	config.DB.Model(&models.Transaction{}).Where("status = ?", "refunded").Count(&totalRefunded)

	// Sum omzet dan profit dari completed transactions
	var completedTransactions []models.Transaction
	config.DB.Preload("Items.Item").Where("status = ?", "completed").Find(&completedTransactions)
	for _, t := range completedTransactions {
		for _, ti := range t.Items {
			totalOmzet += ti.Subtotal
			totalProfit += float64(ti.Quantity) * (ti.Price - ti.Item.BuyPrice)
		}
	}

	// 💰 Hitung profit hari ini
	today := time.Now().Format("2006-01-02")
	var todayTransactions []models.Transaction
	config.DB.Preload("Items.Item").
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Find(&todayTransactions)

	for _, t := range todayTransactions {
		for _, ti := range t.Items {
			todayProfit += float64(ti.Quantity) * (ti.Price - ti.Item.BuyPrice)
		}
	}

	// Low stock count
	var lowStock int64
	config.DB.Model(&models.Item{}).Where("stock < ?", 5).Count(&lowStock)

	// Recent transactions (3 terakhir)
	var recentTransactions []models.Transaction
	config.DB.Preload("Items.Item").Order("created_at desc").Limit(3).Find(&recentTransactions)

	var recentResp []DashboardTransaction
	for _, t := range recentTransactions {
		var items []DashboardTransactionItem
		for _, ti := range t.Items {
			items = append(items, DashboardTransactionItem{
				ItemID:   ti.ItemID,
				Name:     ti.Item.Name,
				Quantity: ti.Quantity,
				Price:    ti.Price,
				Subtotal: ti.Subtotal,
			})
		}
		recentResp = append(recentResp, DashboardTransaction{
			ID:     t.ID,
			Status: t.Status,
			Items:  items,
		})
	}

	fmt.Println("Recent transactions count:", len(recentResp))

	// Top selling items (top 5 berdasarkan total quantity terjual)
	var topItems []TopItem
	config.DB.Model(&models.TransactionItem{}).
		Select("item_id, SUM(quantity) as quantity").
		Joins("JOIN transactions ON transactions.id = transaction_items.transaction_id").
		Where("transactions.status = ?", "completed").
		Group("item_id").
		Order("quantity desc").
		Limit(5).
		Scan(&topItems)

	// Ambil nama item
	for i, ti := range topItems {
		var item models.Item
		if err := config.DB.First(&item, ti.ItemID).Error; err == nil {
			topItems[i].Name = item.Name
		}
	}

	fmt.Println("Top items count:", len(topItems))

	// Kirim response JSON
	c.JSON(http.StatusOK, gin.H{
		"total_items":         totalItems,
		"total_transactions":  totalTransactions,
		"draft":               totalDraft,
		"completed":           totalCompleted,
		"refunded":            totalRefunded,
		"total_omzet":         totalOmzet,
		"total_profit":        totalProfit,
		"today_profit":        todayProfit, 
		"low_stock":           lowStock,
		"recent_transactions": recentResp,
		"top_selling_items":   topItems,
	})
}
