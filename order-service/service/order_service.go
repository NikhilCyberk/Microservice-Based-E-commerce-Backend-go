package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "order-service/order-service/proto"
	productpb "order-service/proto/product"
	userpb "order-service/proto/user"
	"order-service/messaging"
	"gorm.io/gorm"
)

type Order struct {
	ID          string  `gorm:"primaryKey;type:varchar(255)"`
	UserID      string  `gorm:"not null;type:varchar(255);index"`
	ItemsJSON   string  `gorm:"type:text"` // Store items as JSON
	TotalAmount float64 `gorm:"not null;type:decimal(10,2)"`
	Status      string  `gorm:"not null;type:varchar(50);default:'pending'"`
	CreatedAt   int64   `gorm:"autoCreateTime"`
	UpdatedAt   int64   `gorm:"autoUpdateTime"`
}

type OrderService struct {
	pb.UnimplementedOrderServiceServer
	db            *gorm.DB
	userClient    userpb.UserServiceClient
	productClient productpb.ProductServiceClient
	messageBroker *messaging.MessageBroker
}

func NewOrderService(db *gorm.DB, userServiceURL, productServiceURL, rabbitMQURL string) (*OrderService, error) {
    userConn, err := grpc.Dial(userServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to user service: %v", err)
    }

    productConn, err := grpc.Dial(productServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        userConn.Close()
        return nil, fmt.Errorf("failed to connect to product service: %v", err)
    }

	mb, err := messaging.NewMessageBroker(rabbitMQURL)
	if err != nil {
		userConn.Close()
		productConn.Close()
		return nil, fmt.Errorf("failed to connect to message broker: %v", err)
	}

	return &OrderService{
		db:            db,
		userClient:    userpb.NewUserServiceClient(userConn),
		productClient: productpb.NewProductServiceClient(productConn),
		messageBroker: mb,
	}, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	// Verify user exists
	_, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Calculate total and verify inventory
	var totalAmount float64
	for _, item := range req.Items {
		product, err := s.productClient.GetProduct(ctx, &productpb.GetProductRequest{ProductId: item.ProductId})
		if err != nil {
			return nil, fmt.Errorf("product not found: %v", err)
		}

		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %s", item.ProductId)
		}

		totalAmount += product.Price * float64(item.Quantity)

		// Update inventory
		_, err = s.productClient.UpdateInventory(ctx, &productpb.UpdateInventoryRequest{
			ProductId:      item.ProductId,
			QuantityChange: -item.Quantity,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update inventory: %v", err)
		}
	}

	// Convert items to JSON
	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize items: %v", err)
	}

	orderID := generateID()
	order := &Order{
		ID:          orderID,
		UserID:      req.UserId,
		ItemsJSON:   string(itemsJSON),
		TotalAmount: totalAmount,
		Status:      "pending",
	}

	if err := s.db.Create(order).Error; err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Publish order created event
	event := map[string]interface{}{
		"order_id":     orderID,
		"user_id":      req.UserId,
		"total_amount": totalAmount,
		"status":       "pending",
	}
	if err := s.messageBroker.PublishEvent("order_events", "order.created", event); err != nil {
		// Log error but don't fail the order creation
		// In production, you might want to retry or use a transaction
		fmt.Printf("Warning: failed to publish order event: %v\n", err)
	}

	return &pb.OrderResponse{
		OrderId:     orderID,
		UserId:      order.UserID,
		Items:       req.Items,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	var order Order
	result := s.db.Where("id = ?", req.OrderId).First(&order)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	// Deserialize items from JSON
	var items []*pb.OrderItem
	if err := json.Unmarshal([]byte(order.ItemsJSON), &items); err != nil {
		return nil, fmt.Errorf("failed to deserialize items: %v", err)
	}

	return &pb.OrderResponse{
		OrderId:     order.ID,
		UserId:      order.UserID,
		Items:       items,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var dbOrders []Order
	result := s.db.Where("user_id = ?", req.UserId).Find(&dbOrders)
	if result.Error != nil {
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	var orders []*pb.OrderResponse
	for _, o := range dbOrders {
		// Deserialize items from JSON
		var items []*pb.OrderItem
		if err := json.Unmarshal([]byte(o.ItemsJSON), &items); err != nil {
			return nil, fmt.Errorf("failed to deserialize items for order %s: %v", o.ID, err)
		}

		orders = append(orders, &pb.OrderResponse{
			OrderId:     o.ID,
			UserId:      o.UserID,
			Items:       items,
			TotalAmount: o.TotalAmount,
			Status:      o.Status,
		})
	}

	return &pb.ListOrdersResponse{Orders: orders}, nil
}

func generateID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Fallback to timestamp-based ID if random generation fails
        return fmt.Sprintf("%x", b)
    }
    return hex.EncodeToString(b)
}