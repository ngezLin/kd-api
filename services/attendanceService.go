package services

import (
	"errors"
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"time"
)

type AttendanceService interface {
	CreateAttendance(input dtos.CreateAttendanceInput) (*models.Attendance, error)
	GetTodayAttendance() ([]models.Attendance, error)
	GetAttendanceHistory() ([]models.Attendance, error)
	GetAllAttendances() ([]models.Attendance, error)
}

type attendanceService struct{}

func NewAttendanceService() AttendanceService {
	return &attendanceService{}
}

func (s *attendanceService) CreateAttendance(input dtos.CreateAttendanceInput) (*models.Attendance, error) {
	today := time.Now().Truncate(24 * time.Hour)

	// cek absensi hari ini
	var existing models.Attendance
	if err := config.DB.Where("user_id = ? AND date = ?", input.UserID, today).First(&existing).Error; err == nil {
		return nil, errors.New("Cashier sudah diabsen hari ini")
	}

	attendance := models.Attendance{
		UserID: input.UserID,
		Date:   today,
		Status: input.Status,
		Note:   &input.Note,
	}

	if err := config.DB.Create(&attendance).Error; err != nil {
		return nil, err
	}

	// Preload user agar tampil di response
	if err := config.DB.Preload("User").First(&attendance, attendance.ID).Error; err != nil {
		return nil, err
	}

	return &attendance, nil
}

func (s *attendanceService) GetTodayAttendance() ([]models.Attendance, error) {
	var attendances []models.Attendance
	today := time.Now().Truncate(24 * time.Hour)

	if err := config.DB.Preload("User").Where("date = ?", today).Find(&attendances).Error; err != nil {
		return nil, err
	}

	return attendances, nil
}

func (s *attendanceService) GetAttendanceHistory() ([]models.Attendance, error) {
	var attendances []models.Attendance
	if err := config.DB.Preload("User").Order("date DESC").Find(&attendances).Error; err != nil {
		return nil, err
	}
	return attendances, nil
}

func (s *attendanceService) GetAllAttendances() ([]models.Attendance, error) {
	var attendances []models.Attendance

	if err := config.DB.Preload("User").Find(&attendances).Error; err != nil {
		return nil, err
	}

	return attendances, nil
}
