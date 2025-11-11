package main

import (
	"log"
	"net"
	"time"

	pb "product-service/product-service/proto"
	"product-service/service"

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

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProductServiceServer(s, service.NewProductService(db))

	log.Printf("Product service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}