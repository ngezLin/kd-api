package utils

import (
	"kd-api/models"

	"gorm.io/gorm"
)

func CreateTransactionAuditLog(
	db *gorm.DB,
	action string,
	entityID uint,
	oldTx, newTx *models.Transaction,
	userID *uint,
	ipAddress string,
	description string,
) error {
	auditLog := models.AuditLog{
		EntityType:  "transaction",
		EntityID:    entityID,
		Action:      action,
		UserID:      userID,
		OldValue:    toJSONString(oldTx),
		NewValue:    toJSONString(newTx),
		Changes:     calculateTransactionChanges(action, oldTx, newTx),
		IPAddress:   &ipAddress,
		Description: description,
	}

	return db.Create(&auditLog).Error
}

func calculateTransactionChanges(action string, oldTx, newTx *models.Transaction) *string {
	if action != "update" || oldTx == nil || newTx == nil {
		return nil
	}

	changes := make(map[string]interface{})

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

	if getStringValue(oldTx.Note) != getStringValue(newTx.Note) {
		changes["note"] = map[string]string{
			"old": getStringValue(oldTx.Note),
			"new": getStringValue(newTx.Note),
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return toJSONString(changes)
}