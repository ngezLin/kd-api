package utils

import (
	"encoding/json"
	"fmt"
	"kd-api/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ItemResponseCashier struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Stock       int     `json:"stock"`
	Price       float64 `json:"price"`
	ImageURL    *string `json:"image_url,omitempty"`
}

func GetUserRole(c *gin.Context) string {
	role, _ := c.Get("role")
	if role == nil {
		return ""
	}
	return role.(string)
}

func GetUserID(c *gin.Context) *uint {
	keys := []string{"user_id", "userID", "id", "uid", "userId"}
	
	for _, key := range keys {
		if value, exists := c.Get(key); exists {
			switch v := value.(type) {
			case uint:
				return &v
			case int, int64, uint64:
				id := uint(v.(int))
				return &id
			case float64:
				id := uint(v)
				return &id
			}
		}
	}
	return nil
}

func FilterItemsForRole(items []models.Item, role string) interface{} {
	if role != "cashier" {
		return items
	}
	
	cashierItems := make([]ItemResponseCashier, len(items))
	for i, item := range items {
		cashierItems[i] = ItemResponseCashier{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Stock:       item.Stock,
			Price:       item.Price,
			ImageURL:    item.ImageURL,
		}
	}
	return cashierItems
}

func FilterItemForRole(item models.Item, role string) interface{} {
	if role != "cashier" {
		return item
	}
	
	return ItemResponseCashier{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Stock:       item.Stock,
		Price:       item.Price,
		ImageURL:    item.ImageURL,
	}
}

func CreateItemAuditLog(db *gorm.DB, action string, entityID uint, oldItem, newItem *models.Item, userID *uint, ipAddress string, description string) error {
	auditLog := models.AuditLog{
		EntityType:  "item",
		EntityID:    entityID,
		Action:      action,
		UserID:      userID,
		OldValue:    toJSONString(oldItem),
		NewValue:    toJSONString(newItem),
		Changes:     calculateChanges(action, oldItem, newItem),
		IPAddress:   &ipAddress,
		Description: description,
	}
	return db.Create(&auditLog).Error
}

func toJSONString(v interface{}) *string {
	if v == nil {
		return nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	str := string(bytes)
	return &str
}

func calculateChanges(action string, oldItem, newItem *models.Item) *string {
	if action != "update" || oldItem == nil || newItem == nil {
		return nil
	}

	changes := make(map[string]interface{})

	if oldItem.Name != newItem.Name {
		changes["name"] = map[string]string{"old": oldItem.Name, "new": newItem.Name}
	}

	if getStringValue(oldItem.Description) != getStringValue(newItem.Description) {
		changes["description"] = map[string]string{
			"old": getStringValue(oldItem.Description),
			"new": getStringValue(newItem.Description),
		}
	}

	if oldItem.Stock != newItem.Stock {
		changes["stock"] = map[string]int{"old": oldItem.Stock, "new": newItem.Stock}
	}

	if oldItem.BuyPrice != newItem.BuyPrice {
		changes["buy_price"] = map[string]float64{"old": oldItem.BuyPrice, "new": newItem.BuyPrice}
	}

	if oldItem.Price != newItem.Price {
		changes["price"] = map[string]float64{"old": oldItem.Price, "new": newItem.Price}
	}

	if getStringValue(oldItem.ImageURL) != getStringValue(newItem.ImageURL) {
		changes["image_url"] = map[string]string{
			"old": getStringValue(oldItem.ImageURL),
			"new": getStringValue(newItem.ImageURL),
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return toJSONString(changes)
}

func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func FormatItemCSVRow(item models.Item, role string) []string {
	desc := getStringValue(item.Description)
	img := getStringValue(item.ImageURL)

	if role == "cashier" {
		return []string{
			fmt.Sprintf("%d", item.ID),
			item.Name,
			desc,
			fmt.Sprintf("%d", item.Stock),
			fmt.Sprintf("%.2f", item.Price),
			img,
		}
	}

	return []string{
		fmt.Sprintf("%d", item.ID),
		item.Name,
		desc,
		fmt.Sprintf("%d", item.Stock),
		fmt.Sprintf("%.2f", item.BuyPrice),
		fmt.Sprintf("%.2f", item.Price),
		img,
	}
}

func GetCSVHeaders(role string) []string {
	if role == "cashier" {
		return []string{"id", "name", "description", "stock", "price", "image_url"}
	}
	return []string{"id", "name", "description", "stock", "buy_price", "price", "image_url"}
}