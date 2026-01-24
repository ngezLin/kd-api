package config

import (
	"fmt"
	"log"
	"os"

	"kd-api/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	err = db.AutoMigrate(
		&models.Item{},
		&models.Transaction{},
		&models.TransactionItem{},
		&models.User{},
		&models.Attendance{},
		&models.AuditLog{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	DB = db
	fmt.Println("✅ Database connected & migrated successfully")
}
