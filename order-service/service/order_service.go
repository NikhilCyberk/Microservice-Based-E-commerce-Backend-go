package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "order-service/order-service/proto"
	productpb "order-service/proto/product"
	userpb "order-service/proto/user"
	"order-service/messaging"
)

type Order struct {
    ID          string
    UserID      string
    Items       []*pb.OrderItem
    TotalAmount float64
    Status      string
}

type OrderService struct {
    pb.UnimplementedOrderServiceServer
    orders         map[string]*Order
    mu             sync.RWMutex
    userClient     userpb.UserServiceClient
    productClient  productpb.ProductServiceClient
    messageBroker  *messaging.MessageBroker
}

func NewOrderService(userServiceURL, productServiceURL, rabbitMQURL string) (*OrderService, error) {
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
        orders:        make(map[string]*Order),
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

    s.mu.Lock()
    defer s.mu.Unlock()

    orderID := generateID()
    order := &Order{
        ID:          orderID,
        UserID:      req.UserId,
        Items:       req.Items,
        TotalAmount: totalAmount,
        Status:      "pending",
    }

    s.orders[orderID] = order

    // Publish order created event
    event := map[string]interface{}{
        "order_id":     orderID,
        "user_id":      req.UserId,
        "total_amount": totalAmount,
        "status":       "pending",
    }
    if err := s.messageBroker.PublishEvent("orders", "order.created", event); err != nil {
        // Log error but don't fail the order creation
        // In production, you might want to retry or use a transaction
        fmt.Printf("Warning: failed to publish order event: %v\n", err)
    }

    return &pb.OrderResponse{
        OrderId:     orderID,
        UserId:      order.UserID,
        Items:       order.Items,
        TotalAmount: order.TotalAmount,
        Status:      order.Status,
    }, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    order, exists := s.orders[req.OrderId]
    if !exists {
        return nil, fmt.Errorf("order not found")
    }

    return &pb.OrderResponse{
        OrderId:     order.ID,
        UserId:      order.UserID,
        Items:       order.Items,
        TotalAmount: order.TotalAmount,
        Status:      order.Status,
    }, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var orders []*pb.OrderResponse
    for _, o := range s.orders {
        if o.UserID == req.UserId {
            orders = append(orders, &pb.OrderResponse{
                OrderId:     o.ID,
                UserId:      o.UserID,
                Items:       o.Items,
                TotalAmount: o.TotalAmount,
                Status:      o.Status,
            })
        }
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