package dtos

type TopItem struct {
	ItemID   uint   `json:"item_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type DashboardStats struct {
	TodayProfit       float64   `json:"today_profit"`
	TodayTransactions int64     `json:"today_transactions"`
	LowStock          int64     `json:"low_stock"`
	TopSellingItems   []TopItem `json:"top_selling_items"`
}
