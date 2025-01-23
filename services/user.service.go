package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lawfal/go-graph-tripatra/entity"
	"github.com/lawfal/go-graph-tripatra/graph/model"
	"github.com/lawfal/go-graph-tripatra/repository"
	"github.com/lawfal/go-graph-tripatra/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserServiceImpl struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewUserServiceImpl(collection *mongo.Collection, ctx context.Context) repository.UserRepository {
	return &UserServiceImpl{collection, ctx}
}

func (uc *UserServiceImpl) RegisterUser(user *model.RegisterInput) (*entity.UserEntity, error) {
	var newUserEnt entity.RegisterInput
	newUserEnt.CreatedAt = time.Now()
	newUserEnt.UpdatedAt = time.Now()
	newUserEnt.Email = strings.ToLower(user.Email)
	newUserEnt.PasswordConfirm = ""
	newUserEnt.Verified = true
	newUserEnt.Role = "user"
	newUserEnt.Name = user.Name

	hashedPassword, _ := utils.HashPassword(user.Password)
	newUserEnt.Password = hashedPassword

	res, err := uc.collection.InsertOne(uc.ctx, &newUserEnt)

	if err != nil {
		if er, ok := err.(mongo.WriteException); ok && er.WriteErrors[0].Code == 11000 {
			return nil, errors.New("user with that email already exist")
		}
		return nil, err
	}

	opt := options.Index()
	opt.SetUnique(true)
	index := mongo.IndexModel{Keys: bson.M{"email": 1}, Options: opt}

	if _, err := uc.collection.Indexes().CreateOne(uc.ctx, index); err != nil {
		return nil, errors.New("could not create index for email")
	}

	var newUser *entity.UserEntity
	query := bson.M{"_id": res.InsertedID}

	err = uc.collection.FindOne(uc.ctx, query).Decode(&newUser)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (us *UserServiceImpl) FindUserById(id string) (*entity.UserEntity, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var user *entity.UserEntity

	query := bson.M{"_id": objectID}
	err = us.collection.FindOne(us.ctx, query).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}

	return user, nil
}

func (us *UserServiceImpl) FindUserByEmail(email string) (*entity.UserEntity, error) {
	var user *entity.UserEntity

	query := bson.M{"email": strings.ToLower(email)}
	err := us.collection.FindOne(us.ctx, query).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}

	return user, nil
}

func (us *UserServiceImpl) GetAllUser() ([]*entity.UserEntity, error) {
	var result []*entity.UserEntity

	query := bson.M{}
	cursor, err := us.collection.Find(us.ctx, query)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []*entity.UserEntity{}, nil
		}
		return nil, err
	}

	defer cursor.Close(us.ctx)

	for cursor.Next(us.ctx) {
		var user entity.UserEntity
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		result = append(result, &user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (us *UserServiceImpl) UpdateUser(userId string, mod model.UpdateUserInput) (*entity.UserEntity, error) {
	objectID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	updateFields := bson.M{}

	if mod.Name != nil {
		updateFields["name"] = mod.Name
	}
	if mod.Email != nil {
		updateFields["email"] = mod.Email
	}

	update := bson.M{
		"$set": updateFields,
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := us.collection.FindOneAndUpdate(us.ctx, bson.M{"_id": objectID}, update, opts)

	var updatedUser entity.UserEntity
	if err := result.Decode(&updatedUser); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &updatedUser, nil

}

func (us *UserServiceImpl) DeleteUser(userID string) (string, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	result, err := us.collection.DeleteOne(us.ctx, bson.M{"_id": objectID})
	if err != nil {
		return "", err
	}

	if result.DeletedCount == 0 {
		return "", fmt.Errorf("no user found with the provided ID")
	}

	return "User deleted successfully!", nil
}
