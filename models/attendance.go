package models

import (
	"time"

	"gorm.io/gorm"
)

type Attendance struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    UserID    uint           `gorm:"not null;uniqueIndex:unique_user_date" json:"user_id"`
    Date      time.Time      `gorm:"not null;uniqueIndex:unique_user_date" json:"date"`
    Status    string         `gorm:"type:enum('present','absent','off');default:'present'" json:"status"`
    Note      *string        `gorm:"type:text" json:"note,omitempty"`

    User      User           `gorm:"foreignKey:UserID" json:"user"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
