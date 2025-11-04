package main

import (
	"log"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/order-service/handlers"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/order-service/repository"

	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/database"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	dbConfig := database.NewDBConfig()
	db, err := database.ConnectDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&models.Order{}, &models.OrderItem{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repository
	orderRepo := repository.NewOrderRepository(db)

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(orderRepo)

	// Create Gin router
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Routes
	v1 := router.Group("/api/v1")
	{
		orderRoutes := v1.Group("/orders")
		{
			orderRoutes.POST("/", orderHandler.CreateOrder)
			orderRoutes.GET("/:id", orderHandler.GetOrder)
			orderRoutes.PUT("/:id/status", orderHandler.UpdateOrderStatus)
			orderRoutes.GET("/user/:userId", orderHandler.GetUserOrders)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "OK",
			"service": "order-service",
		})
	})

	log.Println("Order service running on :8082")
	router.Run(":8082")
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}