package models

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role" gorm:"type:enum('admin','cashier');default:'cashier'"`
}
