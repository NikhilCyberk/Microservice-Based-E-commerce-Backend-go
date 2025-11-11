package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	pb "product-service/product-service/proto"
	"gorm.io/gorm"
)

type Product struct {
	ID          string  `gorm:"primaryKey;type:varchar(255)"`
	Name        string  `gorm:"not null;type:varchar(255)"`
	Description string  `gorm:"type:text"`
	Price       float64 `gorm:"not null;type:decimal(10,2)"`
	Stock       int32   `gorm:"not null;default:0"`
	CreatedAt   int64   `gorm:"autoCreateTime"`
	UpdatedAt   int64   `gorm:"autoUpdateTime"`
}

type ProductService struct {
	pb.UnimplementedProductServiceServer
	db *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{
		db: db,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	productID := generateID()
	product := &Product{
		ID:          productID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}

	if err := s.db.Create(product).Error; err != nil {
		return nil, fmt.Errorf("failed to create product: %v", err)
	}

	return &pb.ProductResponse{
		ProductId:   productID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}, nil
}

func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	var product Product
	result := s.db.Where("id = ?", req.ProductId).First(&product)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("database error: %v", result.Error)
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
	var dbProducts []Product
	query := s.db

	if req.Limit > 0 {
		query = query.Limit(int(req.Limit))
	}

	result := query.Find(&dbProducts)
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	var products []*pb.ProductResponse
	for _, p := range dbProducts {
		products = append(products, &pb.ProductResponse{
			ProductId:   p.ID,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
			Stock:       p.Stock,
		})
	}

	return &pb.ListProductsResponse{Products: products}, nil
}

func (s *ProductService) UpdateInventory(ctx context.Context, req *pb.UpdateInventoryRequest) (*pb.ProductResponse, error) {
	var product Product
	result := s.db.Where("id = ?", req.ProductId).First(&product)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	newStock := product.Stock + req.QuantityChange
	if newStock < 0 {
		return nil, fmt.Errorf("insufficient stock")
	}

	product.Stock = newStock
	if err := s.db.Save(&product).Error; err != nil {
		return nil, fmt.Errorf("failed to update inventory: %v", err)
	}

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