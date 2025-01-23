package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/lawfal/go-graph-tripatra/config"
	"github.com/lawfal/go-graph-tripatra/graph/model"
	"github.com/lawfal/go-graph-tripatra/repository"
	"github.com/lawfal/go-graph-tripatra/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthController struct {
	UserRepository repository.UserRepository
	RedisClient    *redis.Client
}

func NewAuthController(userRepo repository.UserRepository, redisClient *redis.Client) AuthController {
	return AuthController{userRepo, redisClient}
}

func (ac *AuthController) Register(ctx *gin.Context) {
	var user *model.RegisterInput

	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	if user.Password != user.PasswordConfirm {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "Passwords do not match"})
		return
	}

	newUser, err := ac.UserRepository.RegisterUser(user)

	if err != nil {
		if strings.Contains(err.Error(), "email already exist") {
			ctx.JSON(http.StatusConflict, gin.H{"status": "error", "message": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"status": "success", "data": gin.H{"user": newUser}})
}

func (ac *AuthController) Login(ctx *gin.Context) {
	var credentials *model.LoginInput

	if err := ctx.ShouldBindJSON(&credentials); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	user, err := ac.UserRepository.FindUserByEmail(credentials.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "Invalid email or password"})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	if err := utils.VerifyPassword(user.Password, credentials.Password); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "Invalid email or Password"})
		return
	}

	envConf, _ := config.LoadConfig(".")
	now := time.Now()

	// Generate Tokens
	access_token, err := utils.CreateToken(user.ID.Hex(), envConf.AccessTokenExpiresIn, user.ID, envConf.AccessTokenPrivateKey)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	refresh_token, err := utils.CreateToken(user.ID.Hex(), envConf.RefreshTokenExpiresIn, user.ID, envConf.RefreshTokenPrivateKey)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	// Redis
	errAccess := ac.RedisClient.Set(ctx, access_token.TokenUuid, user.ID.Hex(), time.Unix(*access_token.ExpiresIn, 0).Sub(now)).Err()
	if errAccess != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": errAccess.Error()})
		return
	}

	errRefresh := ac.RedisClient.Set(ctx, refresh_token.TokenUuid, user.ID.Hex(), time.Unix(*refresh_token.ExpiresIn, 0).Sub(now)).Err()
	if errRefresh != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": errRefresh.Error()})
		return
	}

	// Cookies
	ctx.SetCookie("access_token", *access_token.Token, envConf.AccessTokenMaxAge*60, "/", "localhost", false, true)
	ctx.SetCookie("refresh_token", *refresh_token.Token, envConf.RefreshTokenMaxAge*60, "/", "localhost", false, true)
	ctx.SetCookie("logged_in", "true", envConf.AccessTokenMaxAge*60, "/", "localhost", false, false)

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"access_token": access_token, "refresh_token": refresh_token, "profile": user.FilteredResponse()}})
}

func (ac *AuthController) RefreshAccessToken(ctx *gin.Context) {
	message := "could not refresh access token"

	cookie, err := ctx.Cookie("refresh_token")

	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": message})
		return
	}

	envCon, _ := config.LoadConfig(".")

	tokenClaims, err := utils.ValidateToken(cookie, envCon.RefreshTokenPublicKey)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	userid, err := ac.RedisClient.Get(ctx, tokenClaims.TokenUuid).Result()
	if err == redis.Nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": message})
	}

	user, err := ac.UserRepository.FindUserById(fmt.Sprint(userid))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": "the user belonging to this token no longer exists"})
		return
	}

	// Refresh Token
	access_token, err := utils.CreateToken(fmt.Sprint(userid), envCon.AccessTokenExpiresIn, user.ID, envCon.AccessTokenPrivateKey)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	refresh_token, err := utils.CreateToken(fmt.Sprint(userid), envCon.RefreshTokenExpiresIn, user.ID, envCon.RefreshTokenPrivateKey)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	// Refresh Redis
	now := time.Now()
	errAccess := ac.RedisClient.Set(ctx, access_token.TokenUuid, user.ID.Hex(), time.Unix(*access_token.ExpiresIn, 0).Sub(now)).Err()
	if errAccess != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": errAccess.Error()})
		return
	}

	errRefresh := ac.RedisClient.Set(ctx, refresh_token.TokenUuid, user.ID.Hex(), time.Unix(*access_token.ExpiresIn, 0).Sub(now)).Err()
	if errRefresh != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": errRefresh.Error()})
		return
	}

	// Refresh Cookies
	ctx.SetCookie("access_token", *access_token.Token, envCon.AccessTokenMaxAge*60, "/", "localhost", false, true)
	ctx.SetCookie("refresh_token", *refresh_token.Token, envCon.RefreshTokenMaxAge*60, "/", "localhost", false, true)
	ctx.SetCookie("logged_in", "true", envCon.AccessTokenMaxAge*60, "/", "localhost", false, false)

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "access_token": access_token})
}

func (ac *AuthController) Logout(ctx *gin.Context) {
	message := "Token is invalid or session has expired"

	refresh_token, _ := ctx.Cookie("refresh_token")

	if refresh_token == "" {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": message})
		return
	}

	envConf, _ := config.LoadConfig(".")
	tokenClaims, err := utils.ValidateToken(refresh_token, envConf.RefreshTokenPublicKey)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	access_token_uuid, _ := ctx.Get("access_token_uuid")
	if access_token_uuid != nil {
		_, err = ac.RedisClient.Del(ctx, tokenClaims.TokenUuid, access_token_uuid.(string)).Result()
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
			return
		}
	}

	ctx.SetCookie("access_token", "", -1, "/", "localhost", false, true)
	ctx.SetCookie("refresh_token", "", -1, "/", "localhost", false, true)
	ctx.SetCookie("logged_in", "", -1, "/", "localhost", false, true)

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}
