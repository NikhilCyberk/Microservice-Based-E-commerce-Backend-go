package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	pb "product-service/product-service/proto"
)

type Product struct {
    ID          string
    Name        string
    Description string
    Price       float64
    Stock       int32
}

type ProductService struct {
    pb.UnimplementedProductServiceServer
    products map[string]*Product
    mu       sync.RWMutex
}

func NewProductService() *ProductService {
    return &ProductService{
        products: make(map[string]*Product),
    }
}

func (s *ProductService) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    productID := generateID()
    product := &Product{
        ID:          productID,
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
    }

    s.products[productID] = product

    return &pb.ProductResponse{
        ProductId:   productID,
        Name:        product.Name,
        Description: product.Description,
        Price:       product.Price,
        Stock:       product.Stock,
    }, nil
}

func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    product, exists := s.products[req.ProductId]
    if !exists {
        return nil, fmt.Errorf("product not found")
    }

    return &pb.ProductResponse{
        ProductId:   product.ID,
        Name:        product.Name,
        Description: product.Description,
        Price:       product.Price,
        Stock:       product.Stock,
    }, nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var products []*pb.ProductResponse
    count := int32(0)

    for _, p := range s.products {
        if req.Limit > 0 && count >= req.Limit {
            break
        }
        products = append(products, &pb.ProductResponse{
            ProductId:   p.ID,
            Name:        p.Name,
            Description: p.Description,
            Price:       p.Price,
            Stock:       p.Stock,
        })
        count++
    }

    return &pb.ListProductsResponse{Products: products}, nil
}

func (s *ProductService) UpdateInventory(ctx context.Context, req *pb.UpdateInventoryRequest) (*pb.ProductResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    product, exists := s.products[req.ProductId]
    if !exists {
        return nil, fmt.Errorf("product not found")
    }

    newStock := product.Stock + req.QuantityChange
    if newStock < 0 {
        return nil, fmt.Errorf("insufficient stock")
    }

    product.Stock = newStock

    return &pb.ProductResponse{
        ProductId:   product.ID,
        Name:        product.Name,
        Description: product.Description,
        Price:       product.Price,
        Stock:       product.Stock,
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