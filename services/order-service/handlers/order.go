package handlers

import (
	"net/http"
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/services/order-service/repository"

	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	repo repository.OrderRepository
}

func NewOrderHandler(repo repository.OrderRepository) *OrderHandler {
	return &OrderHandler{repo: repo}
}

type CreateOrderRequest struct {
	UserID string              `json:"user_id" binding:"required"`
	Items  []OrderItemRequest `json:"items" binding:"required,min=1"`
}

type OrderItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	Price     float64 `json:"price" binding:"required,min=0"`
}

type UpdateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status" binding:"required,oneof=pending confirmed shipped delivered cancelled"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate user ID
	if _, err := uuid.Parse(req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Convert request items to domain models
	orderItems := make([]models.OrderItem, len(req.Items))
	var totalAmount float64

	for i, item := range req.Items {
		if _, err := uuid.Parse(item.ProductID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
			return
		}

		orderItems[i] = models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
		totalAmount += item.Price * float64(item.Quantity)
	}

	order := &models.Order{
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Status:      models.OrderStatusPending,
		OrderItems:  orderItems,
	}

	if err := h.repo.Create(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"order":   order.ToResponse(),
	})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order.ToResponse())
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	order.Status = req.Status

	if err := h.repo.Update(order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order.ToResponse())
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID := c.Param("userId")
	if _, err := uuid.Parse(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	orders, err := h.repo.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		response[i] = order.ToResponse()
	}

	c.JSON(http.StatusOK, response)
}