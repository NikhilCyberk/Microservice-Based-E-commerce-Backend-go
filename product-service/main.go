package main

import (
	"log"
	"net"

	pb "product-service/product-service/proto"
	"product-service/service"

	"google.golang.org/grpc"
)

func main() {
    lis, err := net.Listen("tcp", ":50052")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    pb.RegisterProductServiceServer(s, service.NewProductService())

    log.Printf("Product service listening on :50052")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}