package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"api-gateway/handlers"
	"api-gateway/middleware"
)

func main() {
	userURL := os.Getenv("USER_SERVICE_URL")
	if userURL == "" {
		userURL = "localhost:50051"
	}

	productURL := os.Getenv("PRODUCT_SERVICE_URL")
	if productURL == "" {
		productURL = "localhost:50052"
	}

	orderURL := os.Getenv("ORDER_SERVICE_URL")
	if orderURL == "" {
		orderURL = "localhost:50053"
	}

	// Wait for services to be ready
	log.Println("Waiting for services to be ready...")
	time.Sleep(10 * time.Second)

	gateway, err := handlers.NewGateway(userURL, productURL, orderURL)
	if err != nil {
		log.Fatal(err)
	}

	// Public routes (no authentication required)
	http.HandleFunc("/users", gateway.CreateUser) // POST /users - Register
	http.HandleFunc("/auth/login", gateway.Login) // POST /auth/login - Login

	// User routes (authentication required)
	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			middleware.AuthMiddleware(gateway.GetUser)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Product routes
	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			// Create product requires admin role
			middleware.RequireRole("admin")(gateway.CreateProduct)(w, r)
		} else if r.Method == "GET" {
			// List products - public access
			gateway.ListProducts(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Get product by ID - public access
	http.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/inventory") {
			// Update inventory requires admin role
			if r.Method == "PUT" {
				middleware.RequireRole("admin")(gateway.UpdateInventory)(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if r.Method == "GET" {
			// Get product - public access
			gateway.GetProduct(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Order routes (authentication required)
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Create order requires authentication
			middleware.AuthMiddleware(gateway.CreateOrder)(w, r)
		} else if r.Method == "GET" {
			// List orders requires authentication
			middleware.AuthMiddleware(gateway.ListOrders)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Get order by ID or update status
	http.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			// Update order status requires admin role
			if r.Method == "PUT" {
				middleware.RequireRole("admin")(gateway.UpdateOrderStatus)(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if r.Method == "GET" {
			// Get order requires authentication
			middleware.AuthMiddleware(gateway.GetOrder)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("API Gateway listening on :8080")
	log.Printf("Connected to User Service: %s", userURL)
	log.Printf("Connected to Product Service: %s", productURL)
	log.Printf("Connected to Order Service: %s", orderURL)
	log.Println("\nAvailable Routes:")
	log.Println("  POST   /users              - Register new user (public)")
	log.Println("  GET    /users/:id          - Get user by ID (auth required)")
	log.Println("  POST   /auth/login         - Login (public)")
	log.Println("  GET    /products           - List products (public)")
	log.Println("  POST   /products            - Create product (admin only)")
	log.Println("  GET    /products/:id        - Get product by ID (public)")
	log.Println("  PUT    /products/:id/inventory - Update inventory (admin only)")
	log.Println("  GET    /orders              - List orders (auth required)")
	log.Println("  POST   /orders              - Create order (auth required)")
	log.Println("  GET    /orders/:id          - Get order by ID (auth required)")
	log.Println("  PUT    /orders/:id/status   - Update order status (admin only)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
