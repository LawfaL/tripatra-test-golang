package repository

import (
	"github.com/lawfal/go-graph-tripatra/entity"
	"github.com/lawfal/go-graph-tripatra/graph/model"
)

type UserRepository interface {
	RegisterUser(*model.RegisterInput) (*entity.UserEntity, error)
	FindUserById(string) (*entity.UserEntity, error)
	FindUserByEmail(string) (*entity.UserEntity, error)
	GetAllUser() ([]*entity.UserEntity, error)
	UpdateUser(string, model.UpdateUserInput) (*entity.UserEntity, error)
	DeleteUser(string) (string, error)
}
