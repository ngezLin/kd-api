package log

import (
	"kd-api/models"
	"kd-api/utils/common"

	"gorm.io/gorm"
)

func CreateAuditLog(
	db *gorm.DB,
	entityType string,
	action string,
	entityID uint,
	oldValue any,
	newValue any,
	changes *string,
	userID *uint,
	ipAddress string,
	description string,
) error {
	auditLog := models.AuditLog{
		EntityType:  entityType,
		EntityID:    entityID,
		Action:      action,
		UserID:      userID,
		OldValue:    common.ToJSONString(oldValue),
		NewValue:    common.ToJSONString(newValue),
		Changes:     changes,
		IPAddress:   &ipAddress,
		Description: description,
	}
	return db.Create(&auditLog).Error
}
