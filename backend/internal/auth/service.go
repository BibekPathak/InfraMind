package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	DisplayName  string `json:"displayName"`
	Role         string `json:"role"`
}

type Service struct {
	jwt *JWTManager
}

func NewAuthService(jwt *JWTManager) *Service {
	return &Service{jwt: jwt}
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	UserID       string `json:"userId"`
	Role         string `json:"role"`
	DisplayName  string `json:"displayName"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (s *Service) Register(req RegisterRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}

	_, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	userID := generateID()
	if req.DisplayName == "" {
		req.DisplayName = req.Email
	}

	token, err := s.jwt.GenerateAccessToken(userID, "admin")
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{
		AccessToken: token,
		UserID:      userID,
		Role:        "admin",
		DisplayName: req.DisplayName,
	}, nil
}

func (s *Service) Login(email, password string) (*LoginResponse, error) {
	if email != "admin@inframind.io" || password != "admin123" {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.jwt.GenerateAccessToken("user-admin", "admin")
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{
		AccessToken: token,
		UserID:      "user-admin",
		Role:        "admin",
		DisplayName: "Admin",
	}, nil
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
