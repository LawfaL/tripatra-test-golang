package graph

import "github.com/lawfal/go-graph-tripatra/repository"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	UserRepository    repository.UserRepository
	ProductRepository repository.ProductRepository
}
