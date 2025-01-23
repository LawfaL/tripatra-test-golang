package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/lawfal/go-graph-tripatra/auth"
	"github.com/lawfal/go-graph-tripatra/config"
	"github.com/lawfal/go-graph-tripatra/graph"
	"github.com/lawfal/go-graph-tripatra/repository"
	"github.com/lawfal/go-graph-tripatra/services"
	"github.com/vektah/gqlparser/v2/ast"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	Mongoclient       *mongo.Client
	RedisClient       *redis.Client
	ctx               context.Context
	server            *gin.Engine
	UserCollection    *mongo.Collection
	ProductCollection *mongo.Collection

	UserRepository    repository.UserRepository
	ProductRepository repository.ProductRepository

	AuthController auth.AuthController
)

func init() {
	envConf, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	ctx = context.TODO()
	Mongoclient = config.ConnectDB(&envConf)
	RedisClient = config.ConnectRedis(&envConf)

	// Collections
	UserCollection = Mongoclient.Database("tripatra-test").Collection("users")
	ProductCollection = Mongoclient.Database("tripatra-test").Collection("products")

	UserRepository = services.NewUserServiceImpl(UserCollection, ctx)
	ProductRepository = services.NewProductServiceImpl(ProductCollection, ctx)

	AuthController = auth.NewAuthController(UserRepository, RedisClient)

	server = gin.Default()
}

func main() {
	envCon, err := config.LoadConfig(".")

	if err != nil {
		log.Fatal("Could not load config", err)
	}

	value, err := RedisClient.Get(ctx, "test").Result()

	if err == redis.Nil {
		fmt.Println("key: test does not exist")
	} else if err != nil {
		panic(err)
	}

	// Cors config
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:8000", "http://localhost:3000", "http://localhost:5173"}
	corsConfig.AllowCredentials = true

	// Graphql config
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{
		UserRepository:    UserRepository,
		ProductRepository: ProductRepository,
	}}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Endpoints
	server.Use(cors.New(corsConfig))

	server.GET("/healthchecker", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": value})
	})

	server.POST("/register", AuthController.Register)
	server.POST("/login", AuthController.Login)

	server.POST("/refresh", AuthController.RefreshAccessToken)
	server.POST("/logout", AuthController.Logout)

	router := server.Use(auth.GetAuth(UserRepository, RedisClient))
	router.GET("/", func(c *gin.Context) {
		playground.Handler("GraphQL playground", "/query").ServeHTTP(c.Writer, c.Request)
	})
	router.POST("/query", func(c *gin.Context) {
		srv.ServeHTTP(c.Writer, c.Request)
	})

	// Log
	defer Mongoclient.Disconnect(ctx)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", envCon.Port)
	log.Fatal(server.Run(":" + envCon.Port))
}
