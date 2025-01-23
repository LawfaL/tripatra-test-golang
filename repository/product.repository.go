package repository

import (
	"github.com/lawfal/go-graph-tripatra/entity"
	"github.com/lawfal/go-graph-tripatra/graph/model"
)

type ProductRepository interface {
	GetAllProduct() ([]*entity.ProductEntity, error)
	FindProductByID(string) (*entity.ProductEntity, error)
	CreateProduct(model.AddProductInput) (*entity.ProductEntity, error)
	UpdateProduct(string, model.UpdateProductInput) (*entity.ProductEntity, error)
	DeleteProduct(string) (string, error)
}
