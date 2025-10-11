package controllers

import (
	"kd-api/config"
	"kd-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Admin: Tambah absensi manual
func CreateAttendance(c *gin.Context) {
    var input struct {
        UserID uint   `json:"user_id" binding:"required"`
        Status string `json:"status" binding:"required,oneof=present absent off"`
        Note   string `json:"note"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    today := time.Now().Truncate(24 * time.Hour)

    // ✅ Cek apakah user sudah diabsen hari ini
    var existing models.Attendance
    if err := config.DB.Where("user_id = ? AND date = ?", input.UserID, today).First(&existing).Error; err == nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cashier sudah diabsen hari ini"})
        return
    }

    attendance := models.Attendance{
        UserID: input.UserID,
        Date:   today,
        Status: input.Status,
        Note:   &input.Note,
    }

    if err := config.DB.Create(&attendance).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Preload user agar tampil di response
    if err := config.DB.Preload("User").First(&attendance, attendance.ID).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, attendance)
}


// Lihat semua absensi hari ini
func GetTodayAttendance(c *gin.Context) {
    var attendances []models.Attendance
    today := time.Now().Truncate(24 * time.Hour)

    if err := config.DB.Preload("User").Where("date = ?", today).Find(&attendances).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, attendances)
}

// Riwayat absensi semua cashier
func GetAttendanceHistory(c *gin.Context) {
    var attendances []models.Attendance
    if err := config.DB.Preload("User").Order("date DESC").Find(&attendances).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, attendances)
}

func GetAttendances(c *gin.Context) {
    var attendances []models.Attendance

    if err := config.DB.Preload("User").Find(&attendances).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, attendances)
}
