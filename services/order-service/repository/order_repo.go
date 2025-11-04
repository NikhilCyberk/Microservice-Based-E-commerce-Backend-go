package repository

import (
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/models"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order *models.Order) error
	GetByID(id string) (*models.Order, error)
	GetByUserID(userID string) ([]models.Order, error)
	Update(order *models.Order) error
	Delete(id string) error
	List(limit, offset int) ([]models.Order, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order *models.Order) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create order
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		
		// Create order items
		for i := range order.OrderItems {
			order.OrderItems[i].OrderID = order.ID
			if err := tx.Create(&order.OrderItems[i]).Error; err != nil {
				return err
			}
		}
		
		return nil
	})
}

func (r *orderRepository) GetByID(id string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("OrderItems").First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) GetByUserID(userID string) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Preload("OrderItems").Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *orderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *orderRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete order items first
		if err := tx.Where("order_id = ?", id).Delete(&models.OrderItem{}).Error; err != nil {
			return err
		}
		// Delete order
		return tx.Delete(&models.Order{}, "id = ?", id).Error
	})
}

func (r *orderRepository) List(limit, offset int) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Preload("OrderItems").Limit(limit).Offset(offset).Order("created_at DESC").Find(&orders).Error
	return orders, err
}