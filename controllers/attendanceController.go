package controllers

import (
	"kd-api/dtos"
	"kd-api/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// tambah absensi manual
func CreateAttendance(c *gin.Context) {
    var input dtos.CreateAttendanceInput

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    service := services.NewAttendanceService()
    attendance, err := service.CreateAttendance(input)

    if err != nil {
        if err.Error() == "Cashier sudah diabsen hari ini" {
             c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
             return
        }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, attendance)
}


// Lihat semua absensi hari ini
func GetTodayAttendance(c *gin.Context) {
    service := services.NewAttendanceService()
    attendances, err := service.GetTodayAttendance()
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, attendances)
}

// Riwayat absensi semua cashier
func GetAttendanceHistory(c *gin.Context) {
    service := services.NewAttendanceService()
    attendances, err := service.GetAttendanceHistory()
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, attendances)
}

func GetAttendances(c *gin.Context) {
    service := services.NewAttendanceService()
    attendances, err := service.GetAllAttendances()

    if err != nil {
         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
         return
    }

    c.JSON(http.StatusOK, attendances)
}
