package log

import (
	"fmt"
	"kd-api/models"
	"kd-api/utils/common"

	"gorm.io/gorm"
)

func CalculateItemChanges(action string, oldItem, newItem *models.Item) *string {
	if action != "update" || oldItem == nil || newItem == nil {
		return nil
	}

	changes := map[string]any{}

	if oldItem.Name != newItem.Name {
		changes["name"] = map[string]string{
			"old": oldItem.Name,
			"new": newItem.Name,
		}
	}

	if common.GetStringValue(oldItem.Description) != common.GetStringValue(newItem.Description) {
		changes["description"] = map[string]string{
			"old": common.GetStringValue(oldItem.Description),
			"new": common.GetStringValue(newItem.Description),
		}
	}

	if oldItem.Stock != newItem.Stock {
		changes["stock"] = map[string]int{
			"old": oldItem.Stock,
			"new": newItem.Stock,
		}
	}

	if oldItem.BuyPrice != newItem.BuyPrice {
		changes["buy_price"] = map[string]float64{
			"old": oldItem.BuyPrice,
			"new": newItem.BuyPrice,
		}
	}

	if oldItem.Price != newItem.Price {
		changes["price"] = map[string]float64{
			"old": oldItem.Price,
			"new": newItem.Price,
		}
	}

	if common.GetStringValue(oldItem.ImageURL) != common.GetStringValue(newItem.ImageURL) {
		changes["image_url"] = map[string]string{
			"old": common.GetStringValue(oldItem.ImageURL),
			"new": common.GetStringValue(newItem.ImageURL),
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return common.ToJSONString(changes)
}

func CreateItemAuditLog(
	db *gorm.DB,
	action string,
	itemID uint,
	oldItem, newItem *models.Item,
	userID *uint,
	ipAddress string,
	description string,
) error {
	var changes *string
	if action == "update" {
		changes = CalculateItemChanges(action, oldItem, newItem)
	}

	return CreateAuditLog(
		db,
		"item",
		action,
		itemID,
		oldItem,
		newItem,
		changes,
		userID,
		ipAddress,
		description,
	)
}

// Optional helper (kalau mau lebih ringkas di controller)
func ItemDescription(action, name string, id uint) string {
	switch action {
	case "create":
		return fmt.Sprintf("Item '%s' created", name)
	case "update":
		return fmt.Sprintf("Item '%s' updated", name)
	case "delete":
		return fmt.Sprintf("Item '%s' deleted", name)
	default:
		return fmt.Sprintf("Item #%d %s", id, action)
	}
}