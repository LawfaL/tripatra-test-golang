package entity

import (
	"time"

	"github.com/lawfal/go-graph-tripatra/graph/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserEntity struct {
	ID              primitive.ObjectID `json:"id" bson:"_id"`
	Name            string             `json:"name" bson:"name"`
	Email           string             `json:"email" bson:"email"`
	Password        string             `json:"password" bson:"password"`
	PasswordConfirm string             `json:"passwordConfirm,omitempty" bson:"passwordConfirm,omitempty"`
	Role            string             `json:"role" bson:"role"`
	Verified        bool               `json:"verified" bson:"verified"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

type RegisterInput struct {
	Name            string    `json:"name" bson:"name"`
	Email           string    `json:"email" bson:"email"`
	Password        string    `json:"password" bson:"password"`
	PasswordConfirm string    `json:"passwordConfirm,omitempty" bson:"passwordConfirm,omitempty"`
	Role            string    `json:"role" bson:"role"`
	Verified        bool      `json:"verified" bson:"verified"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
}

func (user *UserEntity) FilteredResponse() *model.User {
	return &model.User{
		ID:        user.ID.Hex(),
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}
