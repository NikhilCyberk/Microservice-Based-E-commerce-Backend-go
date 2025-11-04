package repository

import (
	"github.com/NikhilCyberk/Microservice-Based-E-commerce-Backend-go/shared/models"

	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(product *models.Product) error
	GetByID(id string) (*models.Product, error)
	GetByIDs(ids []string) ([]models.Product, error)
	Update(product *models.Product) error
	Delete(id string) error
	List(limit, offset int, category string) ([]models.Product, error)
	SearchByName(name string, limit int) ([]models.Product, error)
	GetByCategory(category string, limit, offset int) ([]models.Product, error)
	UpdateStock(productID string, quantity int) error
	UpdateStockBulk(updates map[string]int) error
	GetLowStock(threshold int) ([]models.Product, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *productRepository) GetByID(id string) (*models.Product, error) {
	var product models.Product
	err := r.db.First(&product, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) GetByIDs(ids []string) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("id IN ?", ids).Find(&products).Error
	return products, err
}

func (r *productRepository) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *productRepository) Delete(id string) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error
}

func (r *productRepository) List(limit, offset int, category string) ([]models.Product, error) {
	var products []models.Product
	query := r.db.Model(&models.Product{})
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&products).Error
	return products, err
}

func (r *productRepository) SearchByName(name string, limit int) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("name ILIKE ?", "%"+name+"%").
		Limit(limit).
		Order("name").
		Find(&products).Error
	return products, err
}

func (r *productRepository) GetByCategory(category string, limit, offset int) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("category = ?", category).
		Limit(limit).Offset(offset).
		Order("created_at DESC").
		Find(&products).Error
	return products, err
}

func (r *productRepository) UpdateStock(productID string, quantity int) error {
	return r.db.Model(&models.Product{}).
		Where("id = ?", productID).
		Update("stock", gorm.Expr("stock + ?", quantity)).Error
}

func (r *productRepository) UpdateStockBulk(updates map[string]int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for productID, quantity := range updates {
			if err := tx.Model(&models.Product{}).
				Where("id = ?", productID).
				Update("stock", gorm.Expr("stock + ?", quantity)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *productRepository) GetLowStock(threshold int) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("stock <= ?", threshold).Find(&products).Error
	return products, err
}