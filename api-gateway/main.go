package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"api-gateway/handlers"
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

    http.HandleFunc("/users", gateway.CreateUser)
    http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.Method == "POST" {
            gateway.CreateProduct(w, r)
        } else {
            gateway.ListProducts(w, r)
        }
    })
    http.HandleFunc("/orders", gateway.CreateOrder)

    log.Println("API Gateway listening on :8080")
    log.Printf("Connected to User Service: %s", userURL)
    log.Printf("Connected to Product Service: %s", productURL)
    log.Printf("Connected to Order Service: %s", orderURL)
    log.Fatal(http.ListenAndServe(":8080", nil))
}