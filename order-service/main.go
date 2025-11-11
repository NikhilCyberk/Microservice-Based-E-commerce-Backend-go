package main

import (
	"log"
	"net"
	"os"
	"time"

	pb "order-service/order-service/proto"
	"order-service/service"

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

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "localhost:50051"
	}

	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productServiceURL == "" {
		productServiceURL = "localhost:50052"
	}

	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://admin:admin@localhost:5672/"
	}

	// Wait for dependencies to be ready
	log.Println("Waiting for dependencies to be ready...")
	time.Sleep(5 * time.Second)

	orderService, err := service.NewOrderService(db, userServiceURL, productServiceURL, rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to create order service: %v", err)
	}

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderServiceServer(s, orderService)

	log.Printf("Order service listening on :50053")
	log.Printf("Connected to User Service: %s", userServiceURL)
	log.Printf("Connected to Product Service: %s", productServiceURL)
	log.Printf("Connected to RabbitMQ: %s", rabbitMQURL)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}