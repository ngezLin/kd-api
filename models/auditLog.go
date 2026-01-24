package models

import "time"

type AuditLog struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    EntityType  string    `gorm:"type:varchar(50);not null;index" json:"entity_type"` // "item", "transaction", "user", etc.
    EntityID    uint      `gorm:"not null;index" json:"entity_id"`
    Action      string    `gorm:"type:enum('create','update','delete','status_change');not null" json:"action"`
    UserID      *uint     `gorm:"index" json:"user_id,omitempty"` // Who made the change
    OldValue    *string   `gorm:"type:json" json:"old_value,omitempty"` // JSON of old state
    NewValue    *string   `gorm:"type:json" json:"new_value,omitempty"` // JSON of new state
    Changes     *string   `gorm:"type:json" json:"changes,omitempty"` // Specific fields changed
    IPAddress   *string   `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
    Description string    `gorm:"type:text" json:"description"` // Human-readable description
    CreatedAt   time.Time `gorm:"autoCreateTime;index" json:"created_at"`
    
    // Optional relation
    User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}