package services

import (
	"errors"
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"time"

	"gorm.io/gorm"
)

type CashSessionService interface {
	OpenCashSession(input dtos.OpenCashSessionInput, userID uint) (*models.CashSession, error)
	GetCurrentSession(userID uint) (*models.CashSession, error)
	CloseCashSession(input dtos.CloseCashSessionInput, userID uint) (*models.CashSession, error)
	GetSessionHistory(filter dtos.CashSessionHistoryFilter, userID uint) (*dtos.CashSessionListResponse, error)
}

type cashSessionService struct{}

func NewCashSessionService() CashSessionService {
	return &cashSessionService{}
}

func (s *cashSessionService) OpenCashSession(input dtos.OpenCashSessionInput, userID uint) (*models.CashSession, error) {
	var existing models.CashSession
	err := config.DB.Where("user_id = ? AND status = 'open'", userID).
		First(&existing).Error

	if err == nil {
		return nil, errors.New("cash session masih terbuka")
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	session := models.CashSession{
		UserID:      userID,
		OpeningCash: input.OpeningCash,
		Status:      "open",
		OpenedAt:    time.Now(),
	}

	if err := config.DB.Create(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *cashSessionService) GetCurrentSession(userID uint) (*models.CashSession, error) {
	var session models.CashSession
	if err := config.DB.Where("user_id = ? AND status = 'open'", userID).
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("tidak ada cash session aktif")
		}
		return nil, err
	}
	return &session, nil
}

func (s *cashSessionService) CloseCashSession(input dtos.CloseCashSessionInput, userID uint) (*models.CashSession, error) {
	var session models.CashSession
	if err := config.DB.Where("user_id = ? AND status = 'open'", userID).
		First(&session).Error; err != nil {
		return nil, errors.New("tidak ada cash session terbuka")
	}

	var result struct {
		TotalCashIn float64
		TotalChange float64
	}

	config.DB.Model(&models.Transaction{}).
		Select(
			"COALESCE(SUM(payment), 0) AS total_cash_in, " +
				"COALESCE(SUM(`change`), 0) AS total_change",
		).
		Where(
			"payment_type = ? AND status = ? AND created_at BETWEEN ? AND ?",
			"cash", "completed", session.OpenedAt, time.Now(),
		).
		Scan(&result)

	expected := session.OpeningCash +
		result.TotalCashIn -
		result.TotalChange

	diff := input.ClosingCash - expected

	session.TotalCashIn = result.TotalCashIn
	session.TotalChange = result.TotalChange
	session.ExpectedCash = expected
	session.ClosingCash = &input.ClosingCash
	session.Difference = &diff
	session.Status = "closed"

	now := time.Now()
	session.ClosedAt = &now

	if err := config.DB.Save(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *cashSessionService) GetSessionHistory(filter dtos.CashSessionHistoryFilter, userID uint) (*dtos.CashSessionListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	offset := (filter.Page - 1) * filter.PageSize

	var sessions []models.CashSession
	var total int64

	query := config.DB.Model(&models.CashSession{}).Where("user_id = ?", userID)

	if filter.StartDate != "" {
		query = query.Where("opened_at >= ?", filter.StartDate+" 00:00:00")
	}

	if filter.EndDate != "" {
		query = query.Where("opened_at <= ?", filter.EndDate+" 23:59:59")
	}

	query.Count(&total)

	err := query.Order("opened_at desc").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&sessions).Error

	if err != nil {
		return nil, err
	}

	return &dtos.CashSessionListResponse{
		Data:       sessions,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: int(float64(total)/float64(filter.PageSize) + 0.99),
	}, nil
}
