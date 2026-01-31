package log

import (
	"fmt"
	"kd-api/models"
	"kd-api/utils/common"

	"gorm.io/gorm"
)

func CalculateTransactionChanges(
	action string,
	oldTx, newTx *models.Transaction,
) *string {
	if action != "update" || oldTx == nil || newTx == nil {
		return nil
	}

	changes := map[string]any{}

	if oldTx.Status != newTx.Status {
		changes["status"] = map[string]string{
			"old": oldTx.Status,
			"new": newTx.Status,
		}
	}

	if oldTx.Total != newTx.Total {
		changes["total"] = map[string]float64{
			"old": oldTx.Total,
			"new": newTx.Total,
		}
	}

	if oldTx.Discount != newTx.Discount {
		changes["discount"] = map[string]float64{
			"old": oldTx.Discount,
			"new": newTx.Discount,
		}
	}

	if common.GetStringValue(oldTx.Note) != common.GetStringValue(newTx.Note) {
		changes["note"] = map[string]string{
			"old": common.GetStringValue(oldTx.Note),
			"new": common.GetStringValue(newTx.Note),
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return common.ToJSONString(changes)
}

func CreateTransactionAuditLog(
	db *gorm.DB,
	action string,
	txID uint,
	oldTx, newTx *models.Transaction,
	userID *uint,
	ipAddress string,
	description string,
) error {
	var changes *string
	if action == "update" {
		changes = CalculateTransactionChanges(action, oldTx, newTx)
	}

	return CreateAuditLog(
		db,
		"transaction",
		action,
		txID,
		oldTx,
		newTx,
		changes,
		userID,
		ipAddress,
		description,
	)
}

// Optional helper
func TransactionDescription(action string, id uint) string {
	switch action {
	case "create":
		return fmt.Sprintf("Transaction #%d created", id)
	case "update":
		return fmt.Sprintf("Transaction #%d updated", id)
	case "delete":
		return fmt.Sprintf("Transaction #%d deleted", id)
	default:
		return fmt.Sprintf("Transaction #%d %s", id, action)
	}
}