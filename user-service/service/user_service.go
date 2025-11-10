package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	pb "user-service/user-service/proto"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
    ID           string
    Email        string
    PasswordHash string // Store hashed password
    Name         string
}

type UserService struct {
    pb.UnimplementedUserServiceServer
    users      map[string]*User
    emailIndex map[string]string // email -> userID mapping
    mu         sync.RWMutex
}

func NewUserService() *UserService {
    return &UserService{
        users:      make(map[string]*User),
        emailIndex: make(map[string]string),
    }
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Check if email already exists
    if _, exists := s.emailIndex[req.Email]; exists {
        return nil, fmt.Errorf("email already registered")
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

    s.users[userID] = user
    s.emailIndex[req.Email] = userID

    return &pb.UserResponse{
        UserId: userID,
        Email:  user.Email,
        Name:   user.Name,
    }, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    user, exists := s.users[req.UserId]
    if !exists {
        return nil, fmt.Errorf("user not found")
    }

    return &pb.UserResponse{
        UserId: user.ID,
        Email:  user.Email,
        Name:   user.Name,
    }, nil
}

func (s *UserService) AuthenticateUser(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Find user by email
    userID, exists := s.emailIndex[req.Email]
    if !exists {
        return &pb.AuthResponse{Success: false}, nil
    }

    user := s.users[userID]

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