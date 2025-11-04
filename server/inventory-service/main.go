package main

import (
	"log"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/inventory-service/handlers"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/inventory-service/repository"

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
	err = db.AutoMigrate(&models.Product{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repository
	productRepo := repository.NewProductRepository(db)

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productRepo)

	// Create Gin router
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORSMiddleware())

	// Routes
	v1 := router.Group("/api/v1")
	{
		productRoutes := v1.Group("/products")
		{
			productRoutes.POST("/", productHandler.CreateProduct)
			productRoutes.GET("/", productHandler.ListProducts)
			productRoutes.GET("/:id", productHandler.GetProduct)
			productRoutes.PUT("/:id", productHandler.UpdateProduct)
			productRoutes.DELETE("/:id", productHandler.DeleteProduct)
			productRoutes.PATCH("/:id/stock", productHandler.UpdateStock)
		}
		
		inventoryRoutes := v1.Group("/inventory")
		{
			inventoryRoutes.POST("/check", productHandler.CheckInventory)
			inventoryRoutes.POST("/update-bulk", productHandler.UpdateInventoryBulk)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "OK",
			"service": "inventory-service",
		})
	})

	log.Println("Inventory service running on :8083")
	router.Run(":8083")
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