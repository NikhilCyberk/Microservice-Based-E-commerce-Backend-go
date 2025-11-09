package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	pb "user-service/user-service/proto"
)

type User struct {
    ID       string
    Email    string
    Password string
    Name     string
}

type UserService struct {
    pb.UnimplementedUserServiceServer
    users map[string]*User
    mu    sync.RWMutex
}

func NewUserService() *UserService {
    return &UserService{
        users: make(map[string]*User),
    }
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    userID := generateID()
    user := &User{
        ID:       userID,
        Email:    req.Email,
        Password: req.Password, // In production, hash this!
        Name:     req.Name,
    }

    s.users[userID] = user

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

    for _, user := range s.users {
        if user.Email == req.Email && user.Password == req.Password {
            token := generateID() // Simplified token generation
            return &pb.AuthResponse{
                Success: true,
                UserId:  user.ID,
                Token:   token,
            }, nil
        }
    }

    return &pb.AuthResponse{Success: false}, nil
}

func generateID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}