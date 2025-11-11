package main

import (
	"log"
	"net"
	"time"

	pb "user-service/user-service/proto"
	"user-service/service"

	"google.golang.org/grpc"
)

func main() {
	// Initialize database
	log.Println("Connecting to database...")
	db, err := service.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database connected successfully")

	// Wait a bit for database to be fully ready
	time.Sleep(2 * time.Second)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, service.NewUserService(db))

	log.Printf("User service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}