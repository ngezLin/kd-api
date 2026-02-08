package services

import (
	"errors"
	"kd-api/config"
	"kd-api/dtos"
	"kd-api/models"
	"kd-api/utils"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(input dtos.LoginInput) (*dtos.AuthResponse, error)
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) Login(input dtos.LoginInput) (*dtos.AuthResponse, error) {
	var user models.User
	if err := config.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		return nil, errors.New("User not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, errors.New("Incorrect password")
	}

	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		return nil, errors.New("Failed to generate token")
	}

	return &dtos.AuthResponse{
		Message: "Login successful",
		Token:   token,
		Role:    user.Role,
	}, nil
}
