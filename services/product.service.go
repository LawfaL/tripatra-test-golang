package services

import (
	"context"
	"fmt"
	"time"

	"github.com/lawfal/go-graph-tripatra/entity"
	"github.com/lawfal/go-graph-tripatra/graph/model"
	"github.com/lawfal/go-graph-tripatra/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductServiceImpl struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewProductServiceImpl(collection *mongo.Collection, ctx context.Context) repository.ProductRepository {
	return &ProductServiceImpl{collection, ctx}
}

func (us *ProductServiceImpl) GetAllProduct() ([]*entity.ProductEntity, error) {
	var result []*entity.ProductEntity

	query := bson.M{}
	cursor, err := us.collection.Find(us.ctx, query)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []*entity.ProductEntity{}, nil
		}
		return nil, err
	}

	defer cursor.Close(us.ctx)

	for cursor.Next(us.ctx) {
		var product entity.ProductEntity
		if err := cursor.Decode(&product); err != nil {
			return nil, err
		}
		result = append(result, &product)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (us *ProductServiceImpl) FindProductByID(id string) (*entity.ProductEntity, error) {
	oid, _ := primitive.ObjectIDFromHex(id)

	var user *entity.ProductEntity

	query := bson.M{"_id": oid}
	err := us.collection.FindOne(us.ctx, query).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &entity.ProductEntity{}, err
		}
		return nil, err
	}

	return user, nil
}

func (ps *ProductServiceImpl) CreateProduct(mod model.AddProductInput) (*entity.ProductEntity, error) {
	newProduct := &entity.ProductEntity{
		Name:        mod.Name,
		Description: mod.Description,
		Price:       mod.Price,
		Stock:       int(mod.Stock),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	res, err := ps.collection.InsertOne(ps.ctx, &mod)

	newProduct.ID = res.InsertedID.(primitive.ObjectID)

	if err != nil {
		return nil, err
	}

	return newProduct, nil
}

func (ps *ProductServiceImpl) UpdateProduct(userId string, mod model.UpdateProductInput) (*entity.ProductEntity, error) {
	objectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, fmt.Errorf("invalid product ID format: %w", err)
	}

	_, err = ps.FindProductByID(userId)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	updateFields := bson.M{}

	if mod.Name != nil {
		updateFields["name"] = mod.Name
	}
	if mod.Price != nil {
		updateFields["price"] = mod.Price
	}
	if mod.Stock != nil {
		updateFields["stock"] = mod.Stock
	}

	update := bson.M{
		"$set": updateFields,
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := ps.collection.FindOneAndUpdate(ps.ctx, bson.M{"_id": objectID}, update, opts)

	var updatedProduct entity.ProductEntity
	if err := result.Decode(&updatedProduct); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}

	return &updatedProduct, nil
}

func (us *ProductServiceImpl) DeleteProduct(userID string) (string, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", fmt.Errorf("invalid product ID: %w", err)
	}

	result, err := us.collection.DeleteOne(us.ctx, bson.M{"_id": objectID})
	if err != nil {
		return "", err
	}

	if result.DeletedCount == 0 {
		return "", fmt.Errorf("no product found with the provided ID")
	}

	return "Product deleted successfully!", nil
}
