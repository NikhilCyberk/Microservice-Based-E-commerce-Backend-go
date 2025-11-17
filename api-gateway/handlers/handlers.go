package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    orderpb "api-gateway/proto"
    productpb "api-gateway/proto"
    userpb "api-gateway/proto"
)

type Gateway struct {
    userClient    userpb.UserServiceClient
    productClient productpb.ProductServiceClient
    orderClient   orderpb.OrderServiceClient
}

// createGRPCConnection creates a gRPC connection with load balancing support
// If multiple addresses are provided (comma-separated), it uses round-robin load balancing
func createGRPCConnection(addresses string) (*grpc.ClientConn, error) {
    addrs := strings.Split(addresses, ",")
    for i := range addrs {
        addrs[i] = strings.TrimSpace(addrs[i])
    }

    // If multiple addresses, use round-robin load balancing
    if len(addrs) > 1 {
        // Format: dns:///address1,address2,address3
        addressList := strings.Join(addrs, ",")
        conn, err := grpc.NewClient(
            fmt.Sprintf("dns:///%s", addressList),
            grpc.WithTransportCredentials(insecure.NewCredentials()),
            grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
        )
        return conn, err
    }

    // Single address - direct connection
    conn, err := grpc.NewClient(
        addrs[0],
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    return conn, err
}

func NewGateway(userURL, productURL, orderURL string) (*Gateway, error) {
    userConn, err := createGRPCConnection(userURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to user service: %v", err)
    }

    productConn, err := createGRPCConnection(productURL)
    if err != nil {
        userConn.Close()
        return nil, fmt.Errorf("failed to connect to product service: %v", err)
    }

    orderConn, err := createGRPCConnection(orderURL)
    if err != nil {
        userConn.Close()
        productConn.Close()
        return nil, fmt.Errorf("failed to connect to order service: %v", err)
    }

    return &Gateway{
        userClient:    userpb.NewUserServiceClient(userConn),
        productClient: productpb.NewProductServiceClient(productConn),
        orderClient:   orderpb.NewOrderServiceClient(orderConn),
    }, nil
}

func (g *Gateway) CreateUser(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        Name     string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    resp, err := g.userClient.CreateUser(context.Background(), &userpb.CreateUserRequest{
        Email:    req.Email,
        Password: req.Password,
        Name:     req.Name,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
        return
    }
}

func (g *Gateway) CreateProduct(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name        string  `json:"name"`
        Description string  `json:"description"`
        Price       float64 `json:"price"`
        Stock       int32   `json:"stock"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    resp, err := g.productClient.CreateProduct(context.Background(), &productpb.CreateProductRequest{
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
        return
    }
}

func (g *Gateway) ListProducts(w http.ResponseWriter, r *http.Request) {
    resp, err := g.productClient.ListProducts(context.Background(), &productpb.ListProductsRequest{Limit: 100})
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
        return
    }
}

func (g *Gateway) CreateOrder(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        UserID string `json:"user_id"`
        Items  []struct {
            ProductID string `json:"product_id"`
            Quantity  int32  `json:"quantity"`
        } `json:"items"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
        return
    }

    var items []*orderpb.OrderItem
    for _, item := range req.Items {
        items = append(items, &orderpb.OrderItem{
            ProductId: item.ProductID,
            Quantity:  item.Quantity,
        })
    }

    resp, err := g.orderClient.CreateOrder(context.Background(), &orderpb.CreateOrderRequest{
        UserId: req.UserID,
        Items:  items,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
        return
    }
}