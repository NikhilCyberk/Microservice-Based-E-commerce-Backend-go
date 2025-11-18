package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"api-gateway/middleware"
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

// ========== USER ROUTES ==========

func (g *Gateway) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     req.Role,
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

func (g *Gateway) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL path /users/{id}
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	resp, err := g.userClient.GetUser(context.Background(), &userpb.GetUserRequest{
		UserId: path,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	authResp, err := g.userClient.AuthenticateUser(context.Background(), &userpb.AuthRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !authResp.Success {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	jwtToken, err := middleware.GenerateJWT(authResp.UserId, req.Email, authResp.Role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Get user details
	userResp, err := g.userClient.GetUser(context.Background(), &userpb.GetUserRequest{
		UserId: authResp.UserId,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"token":   jwtToken,
		"user":    userResp,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// ========== PRODUCT ROUTES ==========

func (g *Gateway) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (g *Gateway) GetProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract product ID from URL path /products/{id}
	path := strings.TrimPrefix(r.URL.Path, "/products/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	resp, err := g.productClient.GetProduct(context.Background(), &productpb.GetProductRequest{
		ProductId: path,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
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
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := int32(100)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	resp, err := g.productClient.ListProducts(context.Background(), &productpb.ListProductsRequest{
		Limit: limit,
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

func (g *Gateway) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract product ID from URL path /products/{id}/inventory
	path := strings.TrimPrefix(r.URL.Path, "/products/")
	path = strings.TrimSuffix(path, "/inventory")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		QuantityChange int32 `json:"quantity_change"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := g.productClient.UpdateInventory(context.Background(), &productpb.UpdateInventoryRequest{
		ProductId:      path,
		QuantityChange: req.QuantityChange,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// ========== ORDER ROUTES ==========

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

	// Get user ID from context (set by auth middleware) or use from request
	userID := middleware.GetUserIDFromContext(r)
	if userID != "" {
		req.UserID = userID
	}

	if req.UserID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
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

func (g *Gateway) GetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path /orders/{id}
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserIDFromContext(r)
	userRole := middleware.GetUserRoleFromContext(r)

	resp, err := g.orderClient.GetOrder(context.Background(), &orderpb.GetOrderRequest{
		OrderId: path,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if user can access this order (must be owner or admin)
	if userID != "" && userRole != "admin" && resp.UserId != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) ListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := middleware.GetUserIDFromContext(r)
	userRole := middleware.GetUserRoleFromContext(r)

	// Get user_id from query or use authenticated user
	requestedUserID := r.URL.Query().Get("user_id")
	if requestedUserID == "" {
		requestedUserID = userID
	}

	// Only admins can view other users' orders
	if requestedUserID != userID && userRole != "admin" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if requestedUserID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	resp, err := g.orderClient.ListOrders(context.Background(), &orderpb.ListOrdersRequest{
		UserId: requestedUserID,
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

func (g *Gateway) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path /orders/{id}/status
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	path = strings.TrimSuffix(path, "/status")
	if path == "" || path == r.URL.Path {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := g.orderClient.UpdateOrderStatus(context.Background(), &orderpb.UpdateOrderStatusRequest{
		OrderId: path,
		Status:  req.Status,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
