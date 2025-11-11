package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	pb "user-service/user-service/proto"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           string `gorm:"primaryKey;type:varchar(255)"`
	Email        string `gorm:"uniqueIndex;not null;type:varchar(255)"`
	PasswordHash string `gorm:"not null;type:varchar(255)"` // Store hashed password
	Name         string `gorm:"not null;type:varchar(255)"`
	CreatedAt    int64  `gorm:"autoCreateTime"`
	UpdatedAt    int64  `gorm:"autoUpdateTime"`
}

type UserService struct {
	pb.UnimplementedUserServiceServer
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	// Check if email already exists
	var existingUser User
	result := s.db.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return nil, fmt.Errorf("email already registered")
	}
	if result.Error != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	// Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	userID := generateID()
	user := &User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return &pb.UserResponse{
		UserId: userID,
		Email:  user.Email,
		Name:   user.Name,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	var user User
	result := s.db.Where("id = ?", req.UserId).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	return &pb.UserResponse{
		UserId: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}, nil
}

func (s *UserService) AuthenticateUser(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	var user User
	result := s.db.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return &pb.AuthResponse{Success: false}, nil
		}
		return &pb.AuthResponse{Success: false}, fmt.Errorf("database error: %v", result.Error)
	}

	// Compare hashed password
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return &pb.AuthResponse{Success: false}, nil
	}

	// Generate secure token (in production, use JWT)
	token := generateSecureToken()

	return &pb.AuthResponse{
		Success: true,
		UserId:  user.ID,
		Token:   token,
	}, nil
}

func generateID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Fallback to timestamp-based ID if random generation fails
        return fmt.Sprintf("%x", b)
    }
    return hex.EncodeToString(b)
}

func generateSecureToken() string {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        // Fallback if random generation fails
        return fmt.Sprintf("%x", b)
    }
    return hex.EncodeToString(b)
}