package handlers

import (
	"net/http"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/inventory-service/repository"

	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	repo repository.ProductRepository
}

func NewProductHandler(repo repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Stock       int     `json:"stock" binding:"min=0"`
	Category    string  `json:"category" binding:"required"`
	ImageURL    string  `json:"image_url"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"min=0"`
	Category    string  `json:"category"`
	ImageURL    string  `json:"image_url"`
}

type UpdateStockRequest struct {
	Stock int `json:"stock" binding:"min=0"`
}

type CheckInventoryRequest struct {
	Items []InventoryCheckItem `json:"items" binding:"required,min=1"`
}

type InventoryCheckItem struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type UpdateInventoryBulkRequest struct {
	Updates []InventoryUpdate `json:"updates" binding:"required,min=1"`
}

type InventoryUpdate struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity"` // Can be negative to reduce stock
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		ImageURL:    req.ImageURL,
	}

	if err := h.repo.Create(product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product": product.ToResponse(),
	})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product.ToResponse())
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	limit := 10
	offset := 0

	// TODO: Add pagination parameters
	// optional search/query parameter
	query := c.Query("q")

	products, err := h.repo.List(limit, offset, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]models.ProductResponse, len(products))
	for i, product := range products {
		response[i] = product.ToResponse()
	}

	c.JSON(http.StatusOK, response)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Category != "" {
		product.Category = req.Category
	}
	if req.ImageURL != "" {
		product.ImageURL = req.ImageURL
	}

	if err := h.repo.Update(product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product.ToResponse())
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) UpdateStock(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req UpdateStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	product.Stock = req.Stock

	if err := h.repo.Update(product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product.ToResponse())
}

func (h *ProductHandler) CheckInventory(c *gin.Context) {
	var req CheckInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check inventory for all items
	var missingItems []InventoryCheckItem
	allAvailable := true

	for _, item := range req.Items {
		product, err := h.repo.GetByID(item.ProductID)
		if err != nil || product.Stock < item.Quantity {
			allAvailable = false
			missingItems = append(missingItems, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"available":     allAvailable,
		"missing_items": missingItems,
	})
}

func (h *ProductHandler) UpdateInventoryBulk(c *gin.Context) {
	var req UpdateInventoryBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update inventory for all items
	for _, update := range req.Updates {
		product, err := h.repo.GetByID(update.ProductID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found: " + update.ProductID})
			return
		}

		newStock := product.Stock + update.Quantity
		if newStock < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock for product: " + update.ProductID})
			return
		}

		product.Stock = newStock
		if err := h.repo.Update(product); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inventory updated successfully"})
}