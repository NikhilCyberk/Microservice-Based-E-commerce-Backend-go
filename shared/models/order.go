package models

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID          string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      string      `gorm:"not null;index" json:"user_id"`
	TotalAmount float64     `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	Status      OrderStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	User      User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

type OrderItem struct {
	ID        string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	OrderID   string  `gorm:"not null;index" json:"order_id"`
	ProductID string  `gorm:"not null" json:"product_id"`
	Quantity  int     `gorm:"not null" json:"quantity"`
	Price     float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	
	// Relations
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

type OrderResponse struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	TotalAmount float64         `json:"total_amount"`
	Status      OrderStatus     `json:"status"`
	Items       []OrderItem     `json:"items"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (o *Order) ToResponse() OrderResponse {
	items := make([]OrderItem, len(o.OrderItems))
	// for i, item := range o.OrderItems {
	// 	items[i] = item
	// }
	copy(items, o.OrderItems)

	return OrderResponse{
		ID:          o.ID,
		UserID:      o.UserID,
		TotalAmount: o.TotalAmount,
		Status:      o.Status,
		Items:       items,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}